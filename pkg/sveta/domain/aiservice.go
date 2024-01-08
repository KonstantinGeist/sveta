package domain

import (
	"fmt"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
)

const summaryTriggerThreshold = 2000 // TODO basically disabled for now

type AIService struct {
	agentName         string
	memoryRepository  MemoryRepository
	memoryFactory     MemoryFactory
	responseService   *ResponseService
	promptFormatter   PromptFormatter
	logger            common.Logger
	workingMemorySize int
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
		agentName:         agentName,
		memoryRepository:  memoryRepository,
		memoryFactory:     memoryFactory,
		responseService:   responseService,
		promptFormatter:   promptFormatter,
		logger:            logger,
		workingMemorySize: config.GetIntOrDefault(ConfigKeyWorkingMemorySize, 5),
	}
}

func (a *AIService) Reply(who, what, where string) (string, error) {
	who = strings.TrimSpace(who)
	what = strings.TrimSpace(what)
	where = strings.TrimSpace(where)
	response, err := a.replyImpl(who, what, where)
	if err != nil {
		a.logger.Log(fmt.Sprintf("error responding: %s\n", err.Error()))
		return "...", nil // TODO randomize the answer using the model itself
	}
	return response, nil
}

func (a *AIService) RememberAction(who, what, where string) error {
	memory := a.memoryFactory.NewMemory(MemoryTypeAction, who, what, where)
	return a.memoryRepository.Store(memory)
}

// TODO this is for testing vector search for now, should be rewritten
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

func (a *AIService) ForgetEverything() error {
	return a.memoryRepository.RemoveAll()
}

func (a *AIService) SetContext(context string) error {
	return a.responseService.SetContext(context)
}

func (a *AIService) Summarize(where string) (string, error) {
	memories, err := a.findLatestDialogMemories(where)
	if err != nil {
		return "", err
	}
	return a.responseService.SummarizeMemories(memories)
}

func (a *AIService) replyImpl(who, what, where string) (string, error) {
	err := a.memoryRepository.Store(a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where))
	if err != nil {
		return "", err
	}
	latestMemories, err := a.findLatestDialogMemories(where)
	if err != nil {
		return "", err
	}
	latestSummary, err := a.findLatestSummary(where)
	if err != nil {
		return "", err
	}
	if len(latestMemories) >= summaryTriggerThreshold && (latestSummary == nil || latestSummary.When.Before(latestMemories[0].When)) {
		newSummary, err := a.responseService.SummarizeMemories(MergeMemories(latestMemories, latestSummary))
		if err != nil {
			a.logger.Log(err.Error()) // we can skip summarization if it fails
		}
		if newSummary != "" {
			latestSummary = a.memoryFactory.NewMemory(MemoryTypeSummary, "", newSummary, where)
			err = a.memoryRepository.Store(latestSummary)
			if err != nil {
				return "", err
			}
		}
	}
	recalledMemories, err := a.recallMemoriesFromLongTermMemory(latestMemories)
	if err != nil {
		return "", err
	}
	if len(recalledMemories) > 0 {
		latestMemories = MergeMemories(recalledMemories, latestMemories...)
	}
	response, err := a.responseService.RespondToMemories(MergeMemories(latestMemories, latestSummary))
	if err != nil {
		return "", err
	}
	err = a.memoryRepository.Store(a.memoryFactory.NewMemory(MemoryTypeDialog, a.agentName, response, where))
	if err != nil {
		return "", err
	}
	return response, nil
}

func (a *AIService) findLatestDialogMemories(where string) ([]*Memory, error) {
	notOlderThan := time.Now().Add(-time.Hour)
	return a.memoryRepository.Find(MemoryFilter{
		Types:        []MemoryType{MemoryTypeDialog, MemoryTypeAction},
		Where:        where,
		LatestCount:  a.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
}

func (a *AIService) findLatestSummary(where string) (*Memory, error) {
	latestSummaries, err := a.memoryRepository.Find(MemoryFilter{
		Types:       []MemoryType{MemoryTypeSummary},
		Where:       where,
		LatestCount: 1,
	})
	if err != nil {
		return nil, err
	}
	if len(latestSummaries) == 0 {
		return nil, nil
	}
	return latestSummaries[0], nil
}

func (a *AIService) recallMemoriesFromLongTermMemory(inputMemories []*Memory) ([]*Memory, error) {
	// TODO probably rename it to recallSummaryFromLongTermMemory and return just a single Memory instance
	// TODO must more explicitly tell between fake memory and virtual memory? write about it somewhere in documentation
	if len(inputMemories) == 0 {
		return nil, nil
	}
	inputMemory := inputMemories[len(inputMemories)-1] // let's recall based on the latest memory
	if inputMemory.Embedding == nil {                  // not all memories have embeddings
		return nil, nil
	}
	recalledMemories, err := a.memoryRepository.FindByEmbedding(EmbeddingFilter{
		Where:               inputMemory.Where,
		Embedding:           *inputMemory.Embedding,
		TopCount:            2,                            // TODO can be played with, also can be set with an injected value, like workingMemorySize
		SurroundingCount:    1,                            // TODO can be played with, also can be set with an injected value, like workingMemorySize
		ExcludedIDs:         memoriesToIds(inputMemories), // don't recall what's already in the input
		SimilarityThreshold: 0.1,
	})
	if err != nil {
		return nil, err
	}
	if len(recalledMemories) == 0 {
		return nil, nil
	}
	dialogForLog := a.promptFormatter.FormatDialog(FilterMemoriesByTypes(recalledMemories, []MemoryType{MemoryTypeDialog, MemoryTypeAction}))
	a.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", dialogForLog))
	return recalledMemories, nil
}

func memoriesToIds(memories []*Memory) []string {
	ids := make([]string, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.ID)
	}
	return ids
}
