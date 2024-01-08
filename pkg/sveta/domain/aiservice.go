package domain

import (
	"fmt"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
)

// AIService is the main orchestrator of the whole AI:
// - forms new memories from what is said to the AI, and what is said by the AI itself (via MemoryRepository)
// - recalls similar memories to enrich the context (via MemoryRepository and MemoryFactory)
// - recalls the latest utterances to enrich the context (via MemoryRepository)
// - constructs prompts and retrieves responses from the LLM (large language model) (via PromptFormatter and ResponseService)
type AIService struct {
	agentName                         string
	memoryRepository                  MemoryRepository
	memoryFactory                     MemoryFactory
	responseService                   *ResponseService
	promptFormatter                   PromptFormatter
	logger                            common.Logger
	workingMemorySize                 int
	workingMemoryMaxAge               time.Duration
	episodicMemoryTopCount            int
	episodicMemorySurroundingCount    int
	episodicMemorySimilarityThreshold float64
}

func NewAIService(
	agentName string,
	memoryRepository MemoryRepository,
	memoryFactory MemoryFactory,
	responseService *ResponseService,
	promptFormatter PromptFormatter,
	logger common.Logger,
	config *common.Config,
) *AIService {
	return &AIService{
		agentName:                         agentName,
		memoryRepository:                  memoryRepository,
		memoryFactory:                     memoryFactory,
		responseService:                   responseService,
		promptFormatter:                   promptFormatter,
		logger:                            logger,
		workingMemorySize:                 config.GetIntOrDefault(ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge:               config.GetDurationOrDefault(ConfigKeyWorkingMemoryMaxAge, time.Hour),
		episodicMemoryTopCount:            config.GetIntOrDefault(ConfigKeyEpisodicMemoryTopCount, 2),
		episodicMemorySurroundingCount:    config.GetIntOrDefault(ConfigKeyEpisodicMemorySurroundingCount, 1),
		episodicMemorySimilarityThreshold: config.GetFloatOrDefault(ConfigKeyEpisodicMemorySimilarityThreshold, 0.1),
	}
}

func (a *AIService) Respond(who, what, where string) (string, error) {
	err := a.memoryRepository.Store(a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where))
	if err != nil {
		return "", err
	}
	workingMemories, err := a.recallFromWorkingMemory(where)
	if err != nil {
		return "", err
	}
	episodicMemories, err := a.recallFromEpisodicMemory(workingMemories)
	if err != nil {
		return "", err
	}
	memories := MergeMemories(episodicMemories, workingMemories...)
	response, err := a.responseService.RespondToMemories(memories)
	if err != nil {
		return "", err
	}
	err = a.memoryRepository.Store(a.memoryFactory.NewMemory(MemoryTypeDialog, a.agentName, response, where))
	if err != nil {
		return "", err
	}
	return response, nil
}

func (a *AIService) RememberAction(who, what, where string) error {
	memory := a.memoryFactory.NewMemory(MemoryTypeAction, who, what, where)
	return a.memoryRepository.Store(memory)
}

// LoadMemory TODO this is for testing vector search for now, should be rewritten
func (a *AIService) LoadMemory(path, who, where string, when time.Time) error {
	lines, err := common.ReadLines(path)
	if err != nil {
		return err
	}
	var text string
	for i, line := range lines {
		if i%2 == 0 {
			text = strings.TrimSpace(line)
		} else {
			embedding, err := NewEmbeddingFromFormattedValues(line)
			if err != nil {
				return err
			}
			memory := NewMemory(a.memoryRepository.NextID(), MemoryTypeDialog, who, when, text, where, &embedding)
			err = a.memoryRepository.Store(memory)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// ForgetEverything removes all memory. Useful for debugging.
func (a *AIService) ForgetEverything() error {
	return a.memoryRepository.RemoveAll()
}

// SetContext changes the context. Useful for debugging or when the persona of the AI should change.
func (a *AIService) SetContext(context string) error {
	return a.responseService.SetContext(context)
}

// recallFromWorkingMemory finds memories from the so-called "working memory" -- it's simply N latest memories (depends on
// `workingMemorySize` specified in the context). Working memory is the basis for building proper dialog contexts
// (so that AI could hold continuous dialogs).
func (a *AIService) recallFromWorkingMemory(where string) ([]*Memory, error) {
	// Note that we don't want to recall the latest entries if they're too old (they're most likely already irrelevant).
	notOlderThan := time.Now().Add(-a.workingMemoryMaxAge)
	return a.memoryRepository.Find(MemoryFilter{
		Types:        []MemoryType{MemoryTypeDialog, MemoryTypeAction},
		Where:        where,
		LatestCount:  a.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
}

func (a *AIService) recallFromEpisodicMemory(workingMemories []*Memory) ([]*Memory, error) {
	if len(workingMemories) == 0 {
		return nil, nil
	}
	latestMemory := workingMemories[len(workingMemories)-1] // let's recall based on the latest memory
	if latestMemory.Embedding == nil {                      // not all memories may have embeddings
		return nil, nil
	}
	episodicMemories, err := a.memoryRepository.FindByEmbedding(EmbeddingFilter{
		Where:               latestMemory.Where,
		Embedding:           *latestMemory.Embedding,
		TopCount:            a.episodicMemoryTopCount,
		SurroundingCount:    a.episodicMemorySurroundingCount,
		ExcludedIDs:         GetMemoryIDs(workingMemories), // don't recall what's already in the input
		SimilarityThreshold: a.episodicMemorySimilarityThreshold,
	})
	if err != nil {
		return nil, err
	}
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	dialogForLog := a.promptFormatter.FormatDialog(FilterMemoriesByTypes(episodicMemories, []MemoryType{MemoryTypeDialog, MemoryTypeAction}))
	a.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", dialogForLog))
	return episodicMemories, nil
}
