package notes

type WordFrequencyProvider interface {
	GetPosition(word string) int
}
