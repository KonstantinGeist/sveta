package domain

// StopCondition a model may not have a predetermined stop condition, so this interface allows to configure
// how to stop completion.
type StopCondition interface {
	ShouldStop(prompt, response string) bool
}
