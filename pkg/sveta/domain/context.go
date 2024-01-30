package domain

import (
	"kgeyst.com/sveta/pkg/common"
)

type AIContext struct {
	AgentName        string
	AgentDescription string
}

func NewAIContext(config *common.Config) *AIContext {
	return &AIContext{
		AgentName:        config.GetStringOrDefault(ConfigKeyAgentName, "Sveta"),
		AgentDescription: config.GetString(ConfigKeyAgentDescription),
	}
}
