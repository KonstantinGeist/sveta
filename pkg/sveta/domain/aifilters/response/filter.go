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
	logger common.Logger,
	config *common.Config,
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

func (r *filter) Apply(who, what, where string, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	err := r.memoryRepository.Store(r.memoryFactory.NewMemory(domain.MemoryTypeDialog, who, what, where))
	if err != nil {
		return "", err
	}
	workingMemories, err := r.recallFromWorkingMemory(where)
	if err != nil {
		return "", err
	}
	episodicMemories, err := r.recallFromEpisodicMemory(workingMemories)
	if err != nil {
		return "", err
	}
	memories := domain.MergeMemories(episodicMemories, workingMemories...)
	response, err := r.responseService.RespondToMemoriesWithText(memories, domain.ResponseModeNormal)
	if err != nil {
		return "", err
	}
	err = r.memoryRepository.Store(r.memoryFactory.NewMemory(domain.MemoryTypeDialog, r.aiContext.AgentName, response, where))
	if err != nil {
		return "", err
	}
	return nextAIFilterFunc(r.aiContext.AgentName, response, where)
}

// recallFromWorkingMemory finds memories from the so-called "working memory" -- it's simply N latest memories (depends on
// `workingMemorySize` specified in the context). Working memory is the basis for building proper dialog contexts
// (so that AI could hold continuous dialogs).
func (r *filter) recallFromWorkingMemory(where string) ([]*domain.Memory, error) {
	// Note that we don't want to recall the latest entries if they're too old (they're most likely already irrelevant).
	notOlderThan := time.Now().Add(-r.workingMemoryMaxAge)
	return r.memoryRepository.Find(domain.MemoryFilter{
		Types:        []domain.MemoryType{domain.MemoryTypeDialog, domain.MemoryTypeAction},
		Where:        where,
		LatestCount:  r.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
}

// recallFromEpisodicMemory finds memories in the so-called "episodic memory", or long-term memory which may contain memories from long ago
func (r *filter) recallFromEpisodicMemory(workingMemories []*domain.Memory) ([]*domain.Memory, error) {
	if len(workingMemories) == 0 {
		return nil, nil
	}
	latestMemory := workingMemories[len(workingMemories)-1] // let's recall based on the latest memory
	embeddingsToSearch := r.getHypotheticalEmbeddings(latestMemory.What)
	if latestMemory.Embedding != nil {
		embeddingsToSearch = append(embeddingsToSearch, *latestMemory.Embedding)
	}
	episodicMemories, err := r.memoryRepository.FindByEmbeddings(domain.EmbeddingFilter{
		Where:               latestMemory.Where,
		Embeddings:          embeddingsToSearch,
		TopCount:            r.episodicMemoryFirstStageTopCount,
		SurroundingCount:    r.episodicMemorySurroundingCount,
		ExcludedIDs:         domain.GetMemoryIDs(workingMemories), // don't recall what's already in the input
		SimilarityThreshold: r.episodicMemorySimilarityThreshold,
	})
	if err != nil {
		return nil, err
	}
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	episodicMemories = r.rankMemoriesAndGetTopN(episodicMemories, latestMemory.What, latestMemory.Where)
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	dialogForLog := r.promptFormatterForLog.FormatDialog(domain.FilterMemoriesByTypes(episodicMemories, []domain.MemoryType{domain.MemoryTypeDialog, domain.MemoryTypeAction}))
	r.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", dialogForLog))
	return episodicMemories, nil
}
