package common

import (
	"fmt"
	"strings"
	"time"

	"github.com/dustin/go-humanize"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type AlpacaPromptFormatter struct{}

func NewAlpacaPromptFormatter() *AlpacaPromptFormatter {
	return &AlpacaPromptFormatter{}
}

func (p *AlpacaPromptFormatter) FormatPrompt(options domain.FormatOptions) string {
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
	// the dialog history
	for i := 0; i < len(options.Memories); i++ {
		memory := options.Memories[i]
		buf.WriteString("### ")
		buf.WriteString(memory.Who)
		if !memory.When.IsZero() {
			diff := time.Now().Sub(memory.When)
			if diff > time.Minute {
				buf.WriteString(" (said ")
				buf.WriteString(humanize.Time(memory.When))
				buf.WriteString(")")
			}
		}
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
