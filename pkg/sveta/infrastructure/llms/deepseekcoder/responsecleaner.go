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
	if strings.Contains(options.Prompt, "```") {
		return "" // defense against injection attacks
	}
	const codeMarker = "```"
	response := options.Response
	pythonCodeStartIndex := strings.Index(response, codeMarker)
	if pythonCodeStartIndex == -1 {
		return ""
	}
	response = response[pythonCodeStartIndex+len(codeMarker):]
	pythonCodeEndIndex := strings.Index(response, codeMarker)
	pythonCode := strings.TrimSpace(response[0:pythonCodeEndIndex])
	if strings.HasPrefix(pythonCode, "python") {
		pythonCode = pythonCode[len("python"):]
	}
	return strings.TrimSpace(pythonCode)
}
