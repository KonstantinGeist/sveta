package response

import (
	"fmt"
	"strconv"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

func (r *filter) rankMemoriesAndGetTopN(memories []*domain.Memory, what, where string) []*domain.Memory {
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
	queryMemory := r.memoryFactory.NewMemory(domain.MemoryTypeDialog, "User", query, where)
	response, err := r.getRankerResponseService().RespondToMemoriesWithText([]*domain.Memory{queryMemory}, domain.ResponseModeRerank)
	if err != nil {
		r.logger.Log("failed to rank memories")
		return nil
	}
	indices := r.parseIndicesInRerankerResponse(response, len(memories))
	var result []*domain.Memory
	for _, index := range indices {
		result = append(result, memories[index])
	}
	result = domain.UniqueMemories(result)
	if len(result) == 0 {
		result = memories
	}
	if len(result) > r.episodicMemorySecondStageTopCount {
		result = result[0:r.episodicMemorySecondStageTopCount]
	}
	return result
}

func (r *filter) getRankerResponseService() *domain.ResponseService {
	// TODO internationalize
	rankerAIContext := domain.NewAIContext("RankLLM", "You're RankLLM, an intelligent assistant that can rank passages based on their relevancy to the query.")
	return r.responseService.WithAIContext(rankerAIContext)
}

func (r *filter) formatMemoriesForRanker(memories []*domain.Memory) string {
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

func (r *filter) parseIndicesInRerankerResponse(response string, memoryCount int) []int {
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
