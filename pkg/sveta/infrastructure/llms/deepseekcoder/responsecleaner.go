package deepseekcoder

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type responseCleaner struct{}

func newResponseCleaner() *responseCleaner {
	return &responseCleaner{}
}

func (r *responseCleaner) CleanResponse(options domain.CleanOptions) string {
	return r.removePromptFromResponse(options.Prompt, options.Response)
}

func (r *responseCleaner) removePromptFromResponse(prompt, response string) string {
	if len(response) < len(prompt)+1 {
		return ""
	}
	// The model repeats what was said before, so we remove it from the response.
	return strings.TrimSpace(response[len(prompt)+1:])
}
