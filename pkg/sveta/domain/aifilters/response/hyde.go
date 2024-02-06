package response

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

// getHypotheticalEmbeddings a combination of Hypothetical Document Embeddings (HyDE) + Rewrite-Retrieve-Read
func (r *filter) getHypotheticalEmbeddings(what string) []domain.Embedding {
	if !r.isQuestion(what) { // don't use HyDE for statements -- usually it doesn't work, especially if it's just a casual conversation
		embedding := r.getEmbedding(what)
		if embedding != nil {
			return []domain.Embedding{*embedding}
		}
		return nil
	}
	var output struct {
		Response1 string `json:"response1"`
		Response2 string `json:"response2"`
		Response3 string `json:"response3"`
	}
	err := r.getHyDEResponseService().RespondToQueryWithJSON(
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
	var hypotheticalEmbeddings []domain.Embedding
	for _, response := range hypotheticalResponses {
		embedding := r.getEmbedding(response)
		if embedding != nil {
			hypotheticalEmbeddings = append(hypotheticalEmbeddings, *embedding)
		}
	}
	return hypotheticalEmbeddings
}

func (r *filter) getHyDEResponseService() *domain.ResponseService {
	// TODO internationalize
	hyDEAIContext := domain.NewAIContext("AnswerLLM", "You're AnswerLLM, an intelligent assistant which answers questions to the given user query.")
	return r.responseService.WithAIContext(hyDEAIContext)
}

func (r *filter) isQuestion(what string) bool {
	return strings.Contains(what, "?")
}

func (r *filter) getEmbedding(what string) *domain.Embedding {
	embedding, err := r.embedder.Embed(what)
	if err != nil {
		r.logger.Log(err.Error())
		return nil
	}
	return &embedding
}
