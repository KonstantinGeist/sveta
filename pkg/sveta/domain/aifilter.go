package domain

type NextAIFilterFunc func(context *AIFilterContext) error

// AIFilter an AI agent is internally a chain of "AI filters". An AI filter is able to:
// - modify the input parameters on the fly
// - store useful data for other filters to work with
// - inject additional memories into the global memory repository
// - form the final response
// - pass control to the next AI filters in the chain
// It's similar to filters in Web frameworks. AI filters allow to write modular, extensible code.
type AIFilter interface {
	// Apply implements an AI filter.
	// `nextAIFilterFunc` should always be called when returning from the function (unless we want to stop the chain).
	Apply(context *AIFilterContext, nextAIFilterFunc NextAIFilterFunc) error
}
