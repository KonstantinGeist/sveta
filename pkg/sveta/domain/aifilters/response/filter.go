package response

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type filter struct {
	aiContext                         *domain.AIContext
	memoryFactory                     domain.MemoryFactory
	memoryRepository                  domain.MemoryRepository
	responseService                   *domain.ResponseService
	embedder                          domain.Embedder
	promptFormatterForLog             domain.PromptFormatter
	logger                            common.Logger
	workingMemorySize                 int
	workingMemoryMaxAge               time.Duration
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
		workingMemorySize:                 config.GetIntOrDefault(domain.ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge:               config.GetDurationOrDefault(domain.ConfigKeyWorkingMemoryMaxAge, time.Hour),
		episodicMemoryFirstStageTopCount:  config.GetIntOrDefault(domain.ConfigKeyEpisodicMemoryFirstStageTopCount, 10),
		episodicMemorySecondStageTopCount: config.GetIntOrDefault(domain.ConfigKeyEpisodicMemorySecondStageTopCount, 3),
		episodicMemorySurroundingCount:    config.GetIntOrDefault(domain.ConfigKeyEpisodicMemorySurroundingCount, 1),
		episodicMemorySimilarityThreshold: config.GetFloatOrDefault(domain.ConfigKeyEpisodicMemorySimilarityThreshold, 0.1),
		rankerMaxMemorySize:               config.GetIntOrDefault(domain.ConfigKeyRankerMaxMemorySize, 500),
	}
}

func (f *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	err := f.memoryRepository.Store(f.memoryFactory.NewMemory(domain.MemoryTypeDialog, context.Who, context.What, context.Where))
	if err != nil {
		return "", err
	}
	workingMemories, err := f.recallFromWorkingMemory(context.Where)
	if err != nil {
		return "", err
	}
	episodicMemories, err := f.recallFromEpisodicMemory(workingMemories)
	if err != nil {
		return "", err
	}
	memories := domain.MergeMemories(episodicMemories, workingMemories...)
	response, err := f.responseService.RespondToMemoriesWithText(memories, domain.ResponseModeNormal)
	if err != nil {
		return "", err
	}
	err = f.memoryRepository.Store(f.memoryFactory.NewMemory(domain.MemoryTypeDialog, f.aiContext.AgentName, response, context.Where))
	if err != nil {
		return "", err
	}
	return nextAIFilterFunc(domain.NewAIFilterContext(f.aiContext.AgentName, response, context.Where))
}

// recallFromWorkingMemory finds memories from the so-called "working memory" -- it's simply N latest memories (depends on
// `workingMemorySize` specified in the context). Working memory is the basis for building proper dialog contexts
// (so that AI could hold continuous dialogs).
func (f *filter) recallFromWorkingMemory(where string) ([]*domain.Memory, error) {
	// Note that we don't want to recall the latest entries if they're too old (they're most likely already irrelevant).
	notOlderThan := time.Now().Add(-f.workingMemoryMaxAge)
	return f.memoryRepository.Find(domain.MemoryFilter{
		Types:        []domain.MemoryType{domain.MemoryTypeDialog, domain.MemoryTypeAction},
		Where:        where,
		LatestCount:  f.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
}

// recallFromEpisodicMemory finds memories in the so-called "episodic memory", or long-term memory which may contain memories from long ago
func (f *filter) recallFromEpisodicMemory(workingMemories []*domain.Memory) ([]*domain.Memory, error) {
	if len(workingMemories) == 0 {
		return nil, nil
	}
	rewrittenUserQuery, rewrittenUserQueryEmbedding, err := f.rewriteUserQuery(workingMemories)
	if err != nil {
		return nil, err
	}
	embeddingsToSearch := f.getHypotheticalEmbeddings(rewrittenUserQuery)
	if rewrittenUserQueryEmbedding != nil {
		embeddingsToSearch = append(embeddingsToSearch, *rewrittenUserQueryEmbedding)
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
	episodicMemories = f.rankMemoriesAndGetTopN(episodicMemories, rewrittenUserQuery, where)
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	dialogForLog := f.promptFormatterForLog.FormatDialog(domain.FilterMemoriesByTypes(episodicMemories, []domain.MemoryType{domain.MemoryTypeDialog, domain.MemoryTypeAction}))
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
