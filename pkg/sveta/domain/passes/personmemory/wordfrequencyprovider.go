package personmemory

type WordFrequencyProvider interface {
	GetPosition(word string) int
}
