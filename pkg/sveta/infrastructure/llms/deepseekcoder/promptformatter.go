package deepseekcoder

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type promptFormatter struct{}

func newPromptFormatter() *promptFormatter {
	return &promptFormatter{}
}

func (p *promptFormatter) FormatPrompt(options domain.FormatOptions) string {
	var builder strings.Builder
	builder.WriteString(options.AgentDescription)
	// note that we don't output agent reminder or the summary because it does not support it
	builder.WriteRune('\n')
	for _, memory := range options.Memories {
		if memory.Who == options.AgentName {
			builder.WriteString("### Response:\n")
		} else {
			builder.WriteString("### Instruction:\n")
		}
		builder.WriteString(memory.What)
		builder.WriteRune('\n')
	}
	// forces completion
	builder.WriteString("### Response:\n")
	return builder.String()
}
