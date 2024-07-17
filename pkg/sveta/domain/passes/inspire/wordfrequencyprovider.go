package inspire

type WordFrequencyProvider interface {
	MaxPosition() int
	GetWordAtPosition(position int) string
}
