package domain

import (
	"fmt"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
)

type responseFilter struct {
	aiContext                         *AIContext
	memoryFactory                     MemoryFactory
	memoryRepository                  MemoryRepository
	responseService                   *ResponseService
	promptFormatterForLog             PromptFormatter
	logger                            common.Logger
	workingMemorySize                 int
	workingMemoryMaxAge               time.Duration
	episodicMemoryTopCount            int
	episodicMemorySurroundingCount    int
	episodicMemorySimilarityThreshold float64
}

func NewResponseFilter(
	aiContext *AIContext,
	memoryFactory MemoryFactory,
	memoryRepository MemoryRepository,
	responseService *ResponseService,
	promptFormatterForLog PromptFormatter,
	logger common.Logger,
	config *common.Config,
) AIFilter {
	return &responseFilter{
		aiContext:                         aiContext,
		memoryFactory:                     memoryFactory,
		memoryRepository:                  memoryRepository,
		responseService:                   responseService,
		promptFormatterForLog:             promptFormatterForLog,
		logger:                            logger,
		workingMemorySize:                 config.GetIntOrDefault(ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge:               config.GetDurationOrDefault(ConfigKeyWorkingMemoryMaxAge, time.Hour),
		episodicMemoryTopCount:            config.GetIntOrDefault(ConfigKeyEpisodicMemoryTopCount, 2),
		episodicMemorySurroundingCount:    config.GetIntOrDefault(ConfigKeyEpisodicMemorySurroundingCount, 1),
		episodicMemorySimilarityThreshold: config.GetFloatOrDefault(ConfigKeyEpisodicMemorySimilarityThreshold, 0.1),
	}
}

func (r *responseFilter) Apply(who, what, where string, nextAIFilterFunc NextAIFilterFunc) (string, error) {
	err := r.memoryRepository.Store(r.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where))
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
	memories := MergeMemories(episodicMemories, workingMemories...)
	response, err := r.responseService.RespondToMemoriesWithText(memories)
	if err != nil {
		return "", err
	}
	err = r.memoryRepository.Store(r.memoryFactory.NewMemory(MemoryTypeDialog, r.aiContext.AgentName, response, where))
	if err != nil {
		return "", err
	}
	return nextAIFilterFunc(r.aiContext.AgentName, response, where)
}

// recallFromWorkingMemory finds memories from the so-called "working memory" -- it's simply N latest memories (depends on
// `workingMemorySize` specified in the context). Working memory is the basis for building proper dialog contexts
// (so that AI could hold continuous dialogs).
func (r *responseFilter) recallFromWorkingMemory(where string) ([]*Memory, error) {
	// Note that we don't want to recall the latest entries if they're too old (they're most likely already irrelevant).
	notOlderThan := time.Now().Add(-r.workingMemoryMaxAge)
	return r.memoryRepository.Find(MemoryFilter{
		Types:        []MemoryType{MemoryTypeDialog, MemoryTypeAction},
		Where:        where,
		LatestCount:  r.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
}

func (r *responseFilter) recallFromEpisodicMemory(workingMemories []*Memory) ([]*Memory, error) {
	if len(workingMemories) == 0 {
		return nil, nil
	}
	latestMemory := workingMemories[len(workingMemories)-1] // let's recall based on the latest memory
	embeddingsToSearch := r.getHypotheticalEmbeddings(latestMemory.What)
	if latestMemory.Embedding != nil {
		embeddingsToSearch = append(embeddingsToSearch, *latestMemory.Embedding)
	}
	episodicMemories, err := r.memoryRepository.FindByEmbeddings(EmbeddingFilter{
		Where:               latestMemory.Where,
		Embeddings:          embeddingsToSearch,
		TopCount:            r.episodicMemoryTopCount,
		SurroundingCount:    r.episodicMemorySurroundingCount,
		ExcludedIDs:         GetMemoryIDs(workingMemories), // don't recall what's already in the input
		SimilarityThreshold: r.episodicMemorySimilarityThreshold,
	})
	if err != nil {
		return nil, err
	}
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	dialogForLog := r.promptFormatterForLog.FormatDialog(FilterMemoriesByTypes(episodicMemories, []MemoryType{MemoryTypeDialog, MemoryTypeAction}))
	r.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", dialogForLog))
	return episodicMemories, nil
}

// getHypotheticalEmbeddings an implementation of Hypothetical Document Embeddings (HyDE)
func (r *responseFilter) getHypotheticalEmbeddings(what string) []Embedding {
	if !strings.Contains(what, "?") { // not a question
		memory := r.memoryFactory.NewMemory(MemoryTypeDialog, "", what, "")
		if memory.Embedding != nil {
			return []Embedding{*memory.Embedding}
		}
		return nil
	}
	var output struct {
		Response1 string `json:"response1"`
		Response2 string `json:"response2"`
		Response3 string `json:"response3"`
	}
	// TODO internationalize
	err := r.responseService.RespondToQueryWithJSON("Imagine 3 possible responses to the following user query as if you knew the answer: \""+what+"\"", &output)
	if err != nil {
		r.logger.Log("failed to get hypothetical answers")
		return nil
	}
	var hypotheticalResponses []string
	if output.Response1 != "" {
		hypotheticalResponses = append(hypotheticalResponses, output.Response1)
	}
	if output.Response2 != "" {
		hypotheticalResponses = append(hypotheticalResponses, output.Response2)
	}
	if output.Response3 != "" {
		hypotheticalResponses = append(hypotheticalResponses, output.Response3)
	}
	var hypotheticalEmbeddings []Embedding
	for _, response := range hypotheticalResponses {
		memory := r.memoryFactory.NewMemory(MemoryTypeDialog, "", response, "")
		if memory.Embedding != nil {
			hypotheticalEmbeddings = append(hypotheticalEmbeddings, *memory.Embedding)
		}
	}
	return hypotheticalEmbeddings
}
