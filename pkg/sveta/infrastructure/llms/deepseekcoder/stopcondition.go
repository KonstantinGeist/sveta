package deepseekcoder

type stopCondition struct{}

func newStopCondition() *stopCondition {
	return &stopCondition{}
}

func (s *stopCondition) ShouldStop(_, _ string) bool {
	// Deepseek Coder is good at stopping completion itself
	return false
}
