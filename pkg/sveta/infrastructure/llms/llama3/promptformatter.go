package llama3

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type promptFormatter struct{}

func newPromptFormatter() *promptFormatter {
	return &promptFormatter{}
}

func (p *promptFormatter) FormatPrompt(options domain.FormatOptions) string {
	var buf strings.Builder
	// the system prompt
	buf.WriteString("<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n")
	buf.WriteString(options.AgentDescription)
	// the announced time, if any
	if options.AnnouncedTime != nil {
		buf.WriteString(" Current time is ")
		buf.WriteString(options.AnnouncedTime.Format("Mon, 02 Jan 2006 15:04:05"))
		buf.WriteString(".\n\n")
	}
	if options.JSONOutputSchema != "" {
		buf.WriteString(" Answer using JSON using the following JSON schema: ```\n")
		buf.WriteString(options.JSONOutputSchema)
		buf.WriteString("\n```.\n\n")
	}
	if options.Summary != "" {
		buf.WriteString(options.Summary)
		buf.WriteString("\n\n")
	}
	buf.WriteString("<|eot_id|>")
	// the dialog history
	for i := 0; i < len(options.Memories); i++ {
		memory := options.Memories[i]
		buf.WriteString("<|start_header_id|>")
		buf.WriteString(memory.Who)
		buf.WriteString("<|end_header_id|\n")
		buf.WriteString(memory.What)
		buf.WriteString("<|eot_id|>")
	}
	// the AI agent's hanging name to force the model complete its output
	agentName := options.AgentName
	if options.AgentDescriptionReminder != "" {
		agentName = fmt.Sprintf("%s (%s)", agentName, options.AgentDescriptionReminder)
	}
	buf.WriteString("<|start_header_id|>")
	buf.WriteString(agentName)
	buf.WriteString("<|end_header_id|>\n")
	return buf.String()
}
