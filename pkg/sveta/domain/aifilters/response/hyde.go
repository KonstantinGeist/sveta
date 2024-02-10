package response

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

// getHypotheticalEmbeddings a combination of Hypothetical Document Embeddings (HyDE) + Rewrite-Retrieve-Read
func (f *filter) getHypotheticalEmbeddings(inputMemory *domain.Memory) []domain.Embedding {
	if !f.isQuestion(inputMemory.What) { // don't use HyDE for statements -- usually it doesn't work, especially if it's just a casual conversation
		if inputMemory.Embedding != nil {
			return []domain.Embedding{*inputMemory.Embedding}
		}
		return nil
	}
	var output struct {
		Response1 string `json:"response1"`
		Response2 string `json:"response2"`
		Response3 string `json:"response3"`
	}
	err := f.getHyDEResponseService().RespondToQueryWithJSON(
		"Imagine 3 possible short responses to the following user query as if you knew the answer: \""+inputMemory.What+"\"",
		&output,
	)
	if err != nil {
		f.logger.Log("failed to get hypothetical answers")
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
	var hypotheticalEmbeddings []domain.Embedding
	for _, response := range hypotheticalResponses {
		embedding := f.getEmbedding(response)
		if embedding != nil {
			hypotheticalEmbeddings = append(hypotheticalEmbeddings, *embedding)
		}
	}
	return hypotheticalEmbeddings
}

func (f *filter) getHyDEResponseService() *domain.ResponseService {
	// TODO internationalize
	hyDEAIContext := domain.NewAIContext("AnswerLLM", "You're AnswerLLM, an intelligent assistant which answers questions to the given user query.", "")
	return f.responseService.WithAIContext(hyDEAIContext)
}

func (f *filter) isQuestion(what string) bool {
	return strings.Contains(what, "?")
}
