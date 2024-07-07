package common

import (
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
	// We keep track of how many times the delimiter was found in the output, to understand
	// when we should stop token generation because otherwise the model can continue the dialog forever, and we want
	// to stop as soon as possible.
	return strings.Count(response, "### ") > strings.Count(prompt, "### ")
}
