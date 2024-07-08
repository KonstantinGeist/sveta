package llama3

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type responseCleaner struct{}

func newResponseCleaner() *responseCleaner {
	return &responseCleaner{}
}

func (r *responseCleaner) CleanResponse(options domain.CleanOptions) string {
	prompt := options.Prompt
	response := options.Response
	prompt = strings.ReplaceAll(prompt, "<|begin_of_text|>", "")
	prompt = strings.ReplaceAll(prompt, "<|start_header_id|>", "")
	prompt = strings.ReplaceAll(prompt, "<|end_header_id|>", "")
	prompt = strings.ReplaceAll(prompt, "<|eot_id|>", "")
	if len(response) < len(prompt) {
		return ""
	}
	response = response[len(prompt):]
	return strings.TrimSpace(response)
}
