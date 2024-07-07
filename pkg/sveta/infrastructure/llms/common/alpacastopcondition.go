package common

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type AlpacaStopCondition struct {
	aiContext *domain.AIContext
}

func NewAlpacaStopCondition(aiContext *domain.AIContext) *AlpacaStopCondition {
	return &AlpacaStopCondition{
		aiContext: aiContext,
	}
}

func (a *AlpacaStopCondition) ShouldStop(prompt, response string) bool {
	// We keep track of how many times the agent name with the delimiter was found in the output, to understand
	// when we should stop token generation because otherwise the model can continue the dialog forever, and we want
	// to stop as soon as possible.
	agentNameWithDelimiter := a.getAgentNameWithDelimiter()
	agentNameWithDelimiterAndReminder := a.getAgentNameWithDelimiterAndReminder()
	promptAgentNameCount := strings.Count(prompt, agentNameWithDelimiter) + strings.Count(prompt, agentNameWithDelimiterAndReminder)
	responseAgentNameCount := strings.Count(response, agentNameWithDelimiter) + strings.Count(response, agentNameWithDelimiterAndReminder)
	return responseAgentNameCount > promptAgentNameCount
}

func (a *AlpacaStopCondition) getAgentNameWithDelimiter() string {
	return fmt.Sprintf("### %s:", a.aiContext.AgentName)
}

func (a *AlpacaStopCondition) getAgentNameWithDelimiterAndReminder() string {
	return fmt.Sprintf("### %s (", a.aiContext.AgentName)
}
