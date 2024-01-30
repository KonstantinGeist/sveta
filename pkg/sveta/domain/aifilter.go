package domain

type NextAIFilterFunc func(who, what, where string) (string, error)

// AIFilter an AI agent is internally a chain of "AI filters". An AI filter is able to:
// - modify the parameters on the fly (who, what, etc.)
// - inject additional memories into the global memory repository
// - form the final response
// - pass control to the next AI filters in the chain
// It's similar to filters in Web frameworks. AI filters allow to write modular, extensible code.
type AIFilter interface {
	// Apply implements the AI filter. Parameters `who`, `what` and `where` are identical to the ones in AIService.Respond(..)
	// `nextAIFilterFunc` should always be called when returning from the function (unless we want to stop the chain).
	Apply(who, what, where string, nextAIFilterFunc NextAIFilterFunc) (string, error)
}
