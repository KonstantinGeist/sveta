package domain

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
)

type responseFilter struct {
	aiContext                         *AIContext
	memoryFactory                     MemoryFactory
	memoryRepository                  MemoryRepository
	responseService                   *ResponseService
	embedder                          Embedder
	promptFormatterForLog             PromptFormatter
	logger                            common.Logger
	workingMemorySize                 int
	workingMemoryMaxAge               time.Duration
	episodicMemoryFirstStageTopCount  int
	episodicMemorySecondStageTopCount int
	episodicMemorySurroundingCount    int
	episodicMemorySimilarityThreshold float64
	rankerMaxMemorySize               int
}

func NewResponseFilter(
	aiContext *AIContext,
	memoryFactory MemoryFactory,
	memoryRepository MemoryRepository,
	responseService *ResponseService,
	embedder Embedder,
	promptFormatterForLog PromptFormatter,
	logger common.Logger,
	config *common.Config,
) AIFilter {
	return &responseFilter{
		aiContext:                         aiContext,
		memoryFactory:                     memoryFactory,
		memoryRepository:                  memoryRepository,
		responseService:                   responseService,
		embedder:                          embedder,
		promptFormatterForLog:             promptFormatterForLog,
		logger:                            logger,
		workingMemorySize:                 config.GetIntOrDefault(ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge:               config.GetDurationOrDefault(ConfigKeyWorkingMemoryMaxAge, time.Hour),
		episodicMemoryFirstStageTopCount:  config.GetIntOrDefault(ConfigKeyEpisodicMemoryFirstStageTopCount, 10),
		episodicMemorySecondStageTopCount: config.GetIntOrDefault(ConfigKeyEpisodicMemorySecondStageTopCount, 3),
		episodicMemorySurroundingCount:    config.GetIntOrDefault(ConfigKeyEpisodicMemorySurroundingCount, 1),
		episodicMemorySimilarityThreshold: config.GetFloatOrDefault(ConfigKeyEpisodicMemorySimilarityThreshold, 0.1),
		rankerMaxMemorySize:               config.GetIntOrDefault(ConfigKeyRankerMaxMemorySize, 500),
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

// recallFromEpisodicMemory finds memories in the so-called "episodic memory", or long-term memory which may contain memories from long ago
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
		TopCount:            r.episodicMemoryFirstStageTopCount,
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
	episodicMemories = r.rankMemoriesAndGetTopN(episodicMemories, latestMemory.What, latestMemory.Where)
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	dialogForLog := r.promptFormatterForLog.FormatDialog(FilterMemoriesByTypes(episodicMemories, []MemoryType{MemoryTypeDialog, MemoryTypeAction}))
	r.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", dialogForLog))
	return episodicMemories, nil
}

// getHypotheticalEmbeddings an implementation of Hypothetical Document Embeddings (HyDE)
func (r *responseFilter) getHypotheticalEmbeddings(what string) []Embedding {
	if !r.isQuestion(what) { // don't use HyDE for statements -- usually it doesn't work, especially if it's just a casual conversation
		embedding := r.getEmbedding(what)
		if embedding != nil {
			return []Embedding{*embedding}
		}
		return nil
	}
	var output struct {
		Response1 string `json:"response1"`
		Response2 string `json:"response2"`
		Response3 string `json:"response3"`
	}
	// TODO internationalize
	err := r.responseService.RespondToQueryWithJSON(
		"Imagine 3 possible responses to the following user query as if you knew the answer: \""+what+"\"",
		&output,
	)
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
		embedding := r.getEmbedding(response)
		if embedding != nil {
			hypotheticalEmbeddings = append(hypotheticalEmbeddings, *embedding)
		}
	}
	return hypotheticalEmbeddings
}

func (r *responseFilter) rankMemoriesAndGetTopN(memories []*Memory, what, where string) []*Memory {
	memoriesFormattedForRanker := r.formatMemoriesForRanker(memories)
	// TODO internationalize
	query := fmt.Sprintf(
		"I will provide you with %d passages, each indicated by a numerical identifier [].\nRank the passages based on their relevance to the search query: \"%s\".\n\n%s\nSearch Query: \"%s\".\nRank the %d passages above based on their relevance to the search query. All the passages should be included and listed using identifiers, in descending order of relevance. The output format should be [] > [],\ne.g., [4] > [2]. Only respond with the ranking results, do not say any word or explain.",
		len(memories),
		what,
		memoriesFormattedForRanker,
		what,
		len(memories),
	)
	queryMemory := r.memoryFactory.NewMemory(MemoryTypeDialog, "User", query, where)
	rankerAIContext := NewAIContext("RankLLM", "You are RankLLM, an intelligent assistant that can rank passages based on their relevancy to the query.")
	rankerResponseService := r.responseService.WithAIContext(rankerAIContext)
	response, err := rankerResponseService.RespondToMemoriesWithText([]*Memory{queryMemory})
	if err != nil {
		r.logger.Log("failed to rank memories")
		return nil
	}
	indices := r.parseIndicesInRerankerResponse(response, len(memories))
	var result []*Memory
	for _, index := range indices {
		result = append(result, memories[index])
	}
	result = UniqueMemories(result)
	if len(result) == 0 {
		result = memories
	}
	if len(result) > r.episodicMemorySecondStageTopCount {
		result = result[0:r.episodicMemorySecondStageTopCount]
	}
	return result
}

func (r *responseFilter) formatMemoriesForRanker(memories []*Memory) string {
	var buf strings.Builder
	for index, memory := range memories {
		what := memory.What
		if len(what) > r.rankerMaxMemorySize {
			what = what[0:r.rankerMaxMemorySize] + "..." // trimming it to fit huge memories in the context, at least partially
		}
		buf.WriteString(fmt.Sprintf("[%d] %s\n", index+1, what))
	}
	return buf.String()
}

func (r *responseFilter) parseIndicesInRerankerResponse(response string, memoryCount int) []int {
	response = strings.ReplaceAll(response, "[", "")
	response = strings.ReplaceAll(response, "]", "")
	split := strings.FieldsFunc(response, func(r rune) bool {
		return strings.ContainsRune("><,=", r)
	})
	var indices []int
	for _, s := range split {
		index, err := strconv.Atoi(strings.TrimSpace(s))
		index--
		if err == nil && index >= 0 && index < memoryCount {
			indices = append(indices, index)
		}
	}
	return indices
}

func (r *responseFilter) getEmbedding(what string) *Embedding {
	embedding, err := r.embedder.Embed(what)
	if err != nil {
		r.logger.Log(err.Error())
		return nil
	}
	return &embedding
}

func (r *responseFilter) isQuestion(what string) bool {
	return strings.Contains(what, "?")
}
