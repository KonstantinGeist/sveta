package domain

import (
	"kgeyst.com/sveta/pkg/common"
)

// AIContext the generic info about the AI agent related to the current context: the name and the description (`system prompt`).
type AIContext struct {
	AgentName        string
	AgentDescription string
}

func NewAIContextFromConfig(config *common.Config) *AIContext {
	return &AIContext{
		AgentName:        config.GetStringOrDefault(ConfigKeyAgentName, "Sveta"),
		AgentDescription: config.GetString(ConfigKeyAgentDescription),
	}
}

func NewAIContext(agentName, agentDescription string) *AIContext {
	return &AIContext{
		AgentName:        agentName,
		AgentDescription: agentDescription,
	}
}
