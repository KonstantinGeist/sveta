package domain

import "time"

type FormatOptions struct {
	AgentName                string
	AgentDescription         string
	AgentDescriptionReminder string
	Summary                  string
	AnnouncedTime            *time.Time
	Memories                 []*Memory
	JSONOutputSchema         string
}

type PromptFormatter interface {
	FormatPrompt(options FormatOptions) string
}
