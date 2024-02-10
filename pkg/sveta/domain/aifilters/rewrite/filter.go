package rewrite

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/workingmemory"
)

const DataKeyRewrittenInput = "rewrittenInput"

type filter struct {
	memoryFactory   domain.MemoryFactory
	responseService *domain.ResponseService
	logger          common.Logger
}

func NewFilter(
	memoryFactory domain.MemoryFactory,
	responseService *domain.ResponseService,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		memoryFactory:   memoryFactory,
		responseService: responseService,
		logger:          logger,
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	if len(workingMemories) == 0 { // there's nothing really to rewrite with only 1 working memory
		return nextAIFilterFunc(context)
	}
	var output struct {
		RewrittenUserQuery string `json:"rewrittenUserQuery"`
	}
	memoriesFormattedForRewrite := f.formatMemories(workingMemories, inputMemory.What)
	err := f.getRewriteResponseService().RespondToQueryWithJSON(memoriesFormattedForRewrite, &output)
	if err != nil {
		f.logger.Log(err.Error())
		return nextAIFilterFunc(context)
	}
	rewrittenInputMemory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, inputMemory.Who, output.RewrittenUserQuery, inputMemory.Where)
	return nextAIFilterFunc(context.WithMemory(DataKeyRewrittenInput, rewrittenInputMemory))
}

func (f *filter) getRewriteResponseService() *domain.ResponseService {
	// TODO internationalize
	rankerAIContext := domain.NewAIContext(
		"RewriteLLM",
		"You're RewriteLLM, an intelligent assistant that rewrites a user query to be useful for vector-based search. You must replace pronouns and other ambiguouos words with exact nouns & verbs from the provided chat history. "+
			"For example, if the user says \"I like them\", and previously cats were mentioned, then substitute \"it\" with \"cats\", etc. ",
		"",
	)
	return f.responseService.WithAIContext(rankerAIContext)
}

// TODO internationalize
func (f *filter) formatMemories(workingMemories []*domain.Memory, what string) string {
	var buf strings.Builder
	buf.WriteString("Chat history: ```\n")
	for _, workingMemory := range workingMemories {
		buf.WriteString(fmt.Sprintf("%s: %s\n\n", workingMemory.Who, workingMemory.What))
	}
	buf.WriteString("```\n\n")
	buf.WriteString("Using the chat history above, rewrite the following user query to make it unambiguous: \"")
	buf.WriteString(what)
	buf.WriteString("\"")
	buf.WriteString(" The rewritten query MUST consist of a single short sentence WITHOUT subordinate clauses.")
	return buf.String()
}
