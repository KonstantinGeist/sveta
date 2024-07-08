package response

import (
	"fmt"
	"strconv"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

func (p *pass) rankMemoriesAndGetTopN(context *domain.PassContext, memories []*domain.Memory, what, where string) []*domain.Memory {
	if !context.IsCapabilityEnabled(rerankCapability) {
		return nil
	}
	memoriesFormattedForRanker := p.formatMemoriesForRanker(memories)
	query := fmt.Sprintf(
		"I will provide you with %d passages, each indicated by a numerical identifier [].\nRank the passages based on their relevance to the search query: \"%s\".\n\n%s\nSearch Query: \"%s\".\nRank the %d passages above based on their relevance to the search query. All the passages should be included and listed using identifiers, in descending order of relevance. The output format should be [] > [],\ne.g., [4] > [2]. Only respond with the ranking results, do not say any word or explain.",
		len(memories),
		what,
		memoriesFormattedForRanker,
		what,
		len(memories),
	)
	queryMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, "User", query, where)
	response, err := p.getRankerResponseService().RespondToMemoriesWithText([]*domain.Memory{queryMemory}, domain.ResponseModeRerank)
	if err != nil {
		p.logger.Log("failed to rank memories")
		return nil
	}
	indices := p.parseIndicesInRerankerResponse(response, len(memories))
	var result []*domain.Memory
	for _, index := range indices {
		result = append(result, memories[index])
	}
	result = domain.UniqueMemories(result)
	if len(result) == 0 {
		result = memories
	}
	if len(result) > p.episodicMemorySecondStageTopCount {
		result = result[0:p.episodicMemorySecondStageTopCount]
	}
	return result
}

func (p *pass) getRankerResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext("RankLLM", "You're RankLLM, an intelligent assistant that can rank passages based on their relevancy to the query.", "")
	return p.rerankResponseService.WithAIContext(rankerAIContext)
}

func (p *pass) formatMemoriesForRanker(memories []*domain.Memory) string {
	var buf strings.Builder
	for index, memory := range memories {
		what := memory.What
		if len(what) > p.rerankerMaxMemorySize {
			what = what[0:p.rerankerMaxMemorySize] + "..." // trimming it to fit huge memories in the context, at least partially
		}
		buf.WriteString(fmt.Sprintf("[%d] %s: %s\n", index+1, memory.Who, what))
	}
	return buf.String()
}

func (p *pass) parseIndicesInRerankerResponse(response string, memoryCount int) []int {
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
