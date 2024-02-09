package wiki

type WordFrequencyProvider interface {
	GetPosition(word string) int
}
