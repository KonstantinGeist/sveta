package response

import (
	"fmt"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/memory"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/rewrite"
)

const DataKeyOutput = "output"

type filter struct {
	aiContext                         *domain.AIContext
	memoryFactory                     domain.MemoryFactory
	memoryRepository                  domain.MemoryRepository
	responseService                   *domain.ResponseService
	embedder                          domain.Embedder
	promptFormatterForLog             domain.PromptFormatter
	logger                            common.Logger
	episodicMemoryFirstStageTopCount  int
	episodicMemorySecondStageTopCount int
	episodicMemorySurroundingCount    int
	episodicMemorySimilarityThreshold float64
	rankerMaxMemorySize               int
}

func NewFilter(
	aiContext *domain.AIContext,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	responseService *domain.ResponseService,
	embedder domain.Embedder,
	promptFormatterForLog domain.PromptFormatter,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		aiContext:                         aiContext,
		memoryFactory:                     memoryFactory,
		memoryRepository:                  memoryRepository,
		responseService:                   responseService,
		embedder:                          embedder,
		promptFormatterForLog:             promptFormatterForLog,
		logger:                            logger,
		episodicMemoryFirstStageTopCount:  config.GetIntOrDefault(domain.ConfigKeyEpisodicMemoryFirstStageTopCount, 10),
		episodicMemorySecondStageTopCount: config.GetIntOrDefault(domain.ConfigKeyEpisodicMemorySecondStageTopCount, 3),
		episodicMemorySurroundingCount:    config.GetIntOrDefault(domain.ConfigKeyEpisodicMemorySurroundingCount, 1),
		episodicMemorySimilarityThreshold: config.GetFloatOrDefault(domain.ConfigKeyEpisodicMemorySimilarityThreshold, 0.1),
		rankerMaxMemorySize:               config.GetIntOrDefault(domain.ConfigKeyRankerMaxMemorySize, 500),
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	inputMemory := context.MemoryCoalesced([]string{rewrite.DataKeyRewrittenInput, domain.DataKeyInput})
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	workingMemories := context.Memories(memory.DataKeyWorkingMemory)
	episodicMemories, err := f.recallFromEpisodicMemory(workingMemories, inputMemory)
	if err != nil {
		return err
	}
	memories := domain.MergeMemories(episodicMemories, workingMemories...)
	memories = domain.MergeMemories(memories, inputMemory)
	response, err := f.responseService.RespondToMemoriesWithText(memories, domain.ResponseModeNormal)
	if err != nil {
		return err
	}
	responseMemory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, f.aiContext.AgentName, response, inputMemory.Where)
	return nextAIFilterFunc(context.WithMemory(DataKeyOutput, responseMemory))
}

// recallFromEpisodicMemory finds memories in the so-called "episodic memory", or long-term memory which may contain memories from long ago
func (f *filter) recallFromEpisodicMemory(workingMemories []*domain.Memory, inputMemory *domain.Memory) ([]*domain.Memory, error) {
	if len(workingMemories) == 0 {
		return nil, nil
	}
	embeddingsToSearch := f.getHypotheticalEmbeddings(inputMemory)
	if inputMemory.Embedding != nil {
		embeddingsToSearch = append(embeddingsToSearch, *inputMemory.Embedding)
	}
	where := domain.LastMemory(workingMemories).Where
	episodicMemories, err := f.memoryRepository.FindByEmbeddings(domain.EmbeddingFilter{
		Where:               where,
		Embeddings:          embeddingsToSearch,
		TopCount:            f.episodicMemoryFirstStageTopCount,
		SurroundingCount:    f.episodicMemorySurroundingCount,
		ExcludedIDs:         domain.GetMemoryIDs(workingMemories), // don't recall what's already in the input
		SimilarityThreshold: f.episodicMemorySimilarityThreshold,
	})
	if err != nil {
		return nil, err
	}
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	episodicMemories = f.rankMemoriesAndGetTopN(episodicMemories, inputMemory.What, where)
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	dialogForLog := f.promptFormatterForLog.FormatDialog(domain.FilterMemoriesByTypes(episodicMemories, []domain.MemoryType{domain.MemoryTypeDialog}))
	f.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", dialogForLog))
	return episodicMemories, nil
}

func (f *filter) getEmbedding(what string) *domain.Embedding {
	embedding, err := f.embedder.Embed(what)
	if err != nil {
		f.logger.Log(err.Error())
		return nil
	}
	return &embedding
}
