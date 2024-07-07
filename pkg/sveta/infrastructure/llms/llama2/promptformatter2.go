package llama2

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type PromptFormatter2 struct{}

func NewPromptFormatter2() *PromptFormatter2 {
	return &PromptFormatter2{}
}

func (p *PromptFormatter2) FormatPrompt(options domain.FormatOptions) string {
	var buf strings.Builder
	// the system prompt
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
	// the chat history
	for i := 0; i < len(options.Memories); i++ {
		memory := options.Memories[i]
		buf.WriteString("### ")
		buf.WriteString(memory.Who)
		buf.WriteString(":\n")
		buf.WriteString(memory.What)
		buf.WriteString("\n\n")
	}
	// the AI agent's hanging name to force the model complete its output
	agentName := options.AgentName
	if options.AgentDescriptionReminder != "" {
		agentName = fmt.Sprintf("%s (%s)", agentName, options.AgentDescriptionReminder)
	}
	buf.WriteString("### ")
	buf.WriteString(agentName)
	buf.WriteString(":\n")
	return buf.String()
}
