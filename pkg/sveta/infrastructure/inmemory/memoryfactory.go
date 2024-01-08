package inmemory

import (
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type memoryFactory struct {
	memoryRepository domain.MemoryRepository
	embedder         domain.Embedder
}

func NewMemoryFactory(memoryRepository domain.MemoryRepository, embedder domain.Embedder) domain.MemoryFactory {
	return &memoryFactory{
		memoryRepository: memoryRepository,
		embedder:         embedder,
	}
}

func (m *memoryFactory) NewMemory(typ domain.MemoryType, who string, what string, where string) *domain.Memory {
	return domain.NewMemory(m.memoryRepository.NextID(), typ, who, time.Now(), what, where, m.getSentenceEmbedding(what))
}

func (m *memoryFactory) getSentenceEmbedding(sentence string) *domain.Embedding {
	if sentence == "" {
		return nil
	}
	embedding, err := m.embedder.Embed(sentence)
	if err != nil {
		return nil
	}
	return &embedding
}
