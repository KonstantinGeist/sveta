package response

import (
	"errors"
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

// rewriteUserQuery rewrites the user query (assumed to be the last memory in the provided slice)
// to make it more useful for recall (removes ambiguities, etc.)
func (f *filter) rewriteUserQuery(workingMemories []*domain.Memory) (string, *domain.Embedding, error) {
	if len(workingMemories) == 0 {
		return "", nil, errors.New("empty memory slice")
	}
	if len(workingMemories) == 1 { // there's nothing really to rewrite with only 1 working memory
		return f.rewriteUserQueryFallback(workingMemories)
	}
	var output struct {
		RewrittenUserQuery string `json:"rewrittenUserQuery"`
	}
	memoriesFormattedForRewriter := f.formatMemoriesForRewriter(workingMemories)
	err := f.getRewriterResponseService().RespondToQueryWithJSON(memoriesFormattedForRewriter, &output)
	if err != nil {
		f.logger.Log(err.Error())
		return f.rewriteUserQueryFallback(workingMemories)
	}
	if output.RewrittenUserQuery == "" {
		f.logger.Log("failed to rewrite user query: empty response")
		return f.rewriteUserQueryFallback(workingMemories)
	}
	return output.RewrittenUserQuery, f.getEmbedding(output.RewrittenUserQuery), nil
}

func (f *filter) rewriteUserQueryFallback(workingMemories []*domain.Memory) (string, *domain.Embedding, error) {
	lastMemory := domain.LastMemory(workingMemories)
	return lastMemory.What, lastMemory.Embedding, nil
}

func (f *filter) getRewriterResponseService() *domain.ResponseService {
	// TODO internationalize
	rankerAIContext := domain.NewAIContext(
		"RewriteLLM",
		"You're RewriteLLM, an intelligent assistant that rewrites a user query to be useful for vector-based search. You expand the user query by enriching it with information from the provided chat history. "+
			"For example, if the user says \"I like them\", and previously cats were mentioned, then substitute \"it\" with \"cats\", etc. "+
			"In a nutshell, we want the query to be as unambiguous as possible.",
		"",
	)
	return f.responseService.WithAIContext(rankerAIContext)
}

// TODO internationalize
func (f *filter) formatMemoriesForRewriter(memories []*domain.Memory) string {
	var buf strings.Builder
	buf.WriteString("Chat history: ```\n")
	for _, memory := range memories {
		buf.WriteString(fmt.Sprintf("%s: %s\n", memory.Who, memory.What))
	}
	buf.WriteString("```\n\n")
	buf.WriteString("Using the chat history above, rewrite the following user query to make it unambiguous: \"")
	buf.WriteString(domain.LastMemory(memories).What)
	buf.WriteString("\"")
	return buf.String()
}
