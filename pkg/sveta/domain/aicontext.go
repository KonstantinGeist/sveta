package domain

import (
	"kgeyst.com/sveta/pkg/common"
)

// AIContext the generic info about the AI agent related to the current context: the name and the description (`system prompt`).
type AIContext struct {
	AgentName        string
	AgentDescription string
	// AgentDescriptionReminder see ConfigKeyAgentDescriptionReminder
	AgentDescriptionReminder string
}

func NewAIContextFromConfig(config *common.Config) *AIContext {
	return &AIContext{
		AgentName:                config.GetStringOrDefault(ConfigKeyAgentName, "Sveta"),
		AgentDescription:         config.GetString(ConfigKeyAgentDescription),
		AgentDescriptionReminder: config.GetString(ConfigKeyAgentDescriptionReminder),
	}
}

func NewAIContext(agentName, agentDescription, agentDescriptionReminder string) *AIContext {
	return &AIContext{
		AgentName:                agentName,
		AgentDescription:         agentDescription,
		AgentDescriptionReminder: agentDescriptionReminder,
	}
}
