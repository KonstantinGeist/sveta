package llama3

type stopCondition struct{}

func newStopCondition() *stopCondition {
	return &stopCondition{}
}

func (s *stopCondition) ShouldStop(_, _ string) bool {
	// Llama 3 is good at stopping completion itself
	return false
}
