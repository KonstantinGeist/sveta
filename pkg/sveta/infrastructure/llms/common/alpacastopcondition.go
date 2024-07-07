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
	agentNameWithDelimiter := a.getAgentNameWithDelimiter()
	agentNameWithDelimiterAndReminder := a.getAgentNameWithDelimiterAndReminder()
	return a.shouldStop(prompt, response, agentNameWithDelimiter) ||
		a.shouldStop(prompt, response, agentNameWithDelimiterAndReminder)
}

func (a *AlpacaStopCondition) shouldStop(prompt, response, agentNameDelimiter string) bool {
	// We keep track of how many times the agent name with the delimiter was found in the output, to understand
	// when we should stop token generation because otherwise the model can continue the dialog forever, and we want
	// to stop as soon as possible.
	promptAgentNameCount := strings.Count(prompt, agentNameDelimiter)
	responseAgentNameCount := strings.Count(response, agentNameDelimiter)
	return responseAgentNameCount > promptAgentNameCount
}

func (a *AlpacaStopCondition) getAgentNameWithDelimiter() string {
	return fmt.Sprintf("### %s:", a.aiContext.AgentName)
}

func (a *AlpacaStopCondition) getAgentNameWithDelimiterAndReminder() string {
	return fmt.Sprintf("### %s (", a.aiContext.AgentName)
}
