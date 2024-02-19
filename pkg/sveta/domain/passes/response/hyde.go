package response

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

// getHypotheticalEmbeddings a combination of Hypothetical Document Embeddings (HyDE) + Rewrite-Retrieve-Read
func (p *pass) getHypotheticalEmbeddings(context *domain.PassContext, inputMemory *domain.Memory) []domain.Embedding {
	if !context.IsCapabilityEnabled(hydeCapability) {
		return nil
	}
	if !p.isQuestion(inputMemory.What) { // don't use HyDE for statements -- usually it doesn't work, especially if it's just a casual conversation
		if inputMemory.Embedding != nil {
			return []domain.Embedding{*inputMemory.Embedding}
		}
		return nil
	}
	var output struct {
		Response1 string `json:"response1"`
		Response2 string `json:"response2"`
	}
	err := p.getHyDEResponseService().RespondToQueryWithJSON(
		"Imagine 2 possible short responses to the following user query as if you knew the answer: \""+inputMemory.What+"\"",
		&output,
	)
	if err != nil {
		p.logger.Log("failed to get hypothetical answers")
		return nil
	}
	var hypotheticalResponses []string
	if output.Response1 != "" {
		hypotheticalResponses = append(hypotheticalResponses, output.Response1)
	}
	if output.Response2 != "" {
		hypotheticalResponses = append(hypotheticalResponses, output.Response2)
	}
	var hypotheticalEmbeddings []domain.Embedding
	for _, response := range hypotheticalResponses {
		embedding := p.getEmbedding(response)
		if embedding != nil {
			hypotheticalEmbeddings = append(hypotheticalEmbeddings, *embedding)
		}
	}
	return hypotheticalEmbeddings
}

func (p *pass) getHyDEResponseService() *domain.ResponseService {
	hyDEAIContext := domain.NewAIContext("AnswerLLM", "You're AnswerLLM, an intelligent assistant which answers questions to the given user query.", "")
	return p.responseService.WithAIContext(hyDEAIContext)
}

func (p *pass) isQuestion(what string) bool {
	return strings.Contains(what, "?")
}
