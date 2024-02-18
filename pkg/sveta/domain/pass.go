package domain

type NextPassFunc func(context *PassContext) error

// Pass an AI agent is internally a chain of "passes". A pass is able to:
// - modify the input parameters on the fly
// - store useful data for other passes to work with
// - inject additional memories into the global memory repository
// - form the final response
// - pass control to the next passes in the chain
// It's similar to passes in Web frameworks. Passes allow to write modular, extensible code.
type Pass interface {
	Capabilities() []*Capability
	// Apply implements a pass.
	// `nextPassFunc` should always be called when returning from the function (unless we want to stop the chain).
	Apply(context *PassContext, nextPassFunc NextPassFunc) error
}
