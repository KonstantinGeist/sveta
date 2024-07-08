package rewrite

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/workingmemory"
)

const DataKeyRewrittenInput = "rewrittenInput"

const rewriteCapability = "rewrite"

type pass struct {
	memoryFactory   domain.MemoryFactory
	responseService *domain.ResponseService
	logger          common.Logger
}

func NewPass(
	memoryFactory domain.MemoryFactory,
	responseService *domain.ResponseService,
	logger common.Logger,
) domain.Pass {
	return &pass{
		memoryFactory:   memoryFactory,
		responseService: responseService,
		logger:          logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        rewriteCapability,
			Description: "rewrites the user query to make it less ambiguous",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(rewriteCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	if len(workingMemories) == 0 { // there's nothing really to rewrite with only 1 working memory
		return nextPassFunc(context)
	}
	var output struct {
		RewrittenUserQuery string `json:"rewrittenUserQuery"`
	}
	memoriesFormattedForRewrite := p.formatMemories(workingMemories, inputMemory.What)
	err := p.getRewriteResponseService().RespondToQueryWithJSON(memoriesFormattedForRewrite, &output)
	if err != nil {
		p.logger.Log(err.Error())
		return nextPassFunc(context)
	}
	rewrittenInputMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, inputMemory.Who, output.RewrittenUserQuery, inputMemory.Where)
	return nextPassFunc(context.WithMemory(DataKeyRewrittenInput, rewrittenInputMemory))
}

func (p *pass) getRewriteResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext(
		"RewriteLLM",
		"You're RewriteLLM, an intelligent assistant that rewrites a user query to be useful for vector-based search. You must replace pronouns and other ambiguouos words with exact nouns & verbs from the provided chat history. "+
			"For example, if the user says \"I like them\", and previously cats were mentioned, then substitute \"it\" with \"cats\", etc. ",
		"",
	)
	return p.responseService.WithAIContext(rankerAIContext)
}

func (p *pass) formatMemories(workingMemories []*domain.Memory, what string) string {
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
