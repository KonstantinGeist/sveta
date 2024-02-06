package domain

type MemoryRepository interface {
	NextID() string // should be time-sortable
	Store(memory *Memory) error
	Find(filter MemoryFilter) ([]*Memory, error)
	FindByEmbeddings(filter EmbeddingFilter) ([]*Memory, error)
	RemoveAll() error
}
