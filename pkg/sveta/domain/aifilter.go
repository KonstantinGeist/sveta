package domain

type NextAIFilterFunc func(context AIFilterContext) (string, error)

type AIFilterContext struct {
	Who   string
	What  string
	Where string
}

// AIFilter an AI agent is internally a chain of "AI filters". An AI filter is able to:
// - modify the parameters on the fly (who, what, etc.)
// - inject additional memories into the global memory repository
// - form the final response
// - pass control to the next AI filters in the chain
// It's similar to filters in Web frameworks. AI filters allow to write modular, extensible code.
type AIFilter interface {
	// Apply implements the AI filter. Parameters `who`, `what` and `where` (see `context`) are identical to the ones in AIService.Respond(..)
	// `nextAIFilterFunc` should always be called when returning from the function (unless we want to stop the chain).
	Apply(context AIFilterContext, nextAIFilterFunc NextAIFilterFunc) (string, error)
}

func NewAIFilterContext(who, what, where string) AIFilterContext {
	return AIFilterContext{
		Who:   who,
		What:  what,
		Where: where,
	}
}

func (a AIFilterContext) WithWhat(what string) AIFilterContext {
	a.What = what
	return a
}
