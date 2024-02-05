package inmemory

import (
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type MemoryFactory struct {
	memoryRepository domain.MemoryRepository
	embedder         domain.Embedder
}

func NewMemoryFactory(memoryRepository domain.MemoryRepository, embedder domain.Embedder) *MemoryFactory {
	return &MemoryFactory{
		memoryRepository: memoryRepository,
		embedder:         embedder,
	}
}

func (m *MemoryFactory) NewMemory(typ domain.MemoryType, who string, what string, where string) *domain.Memory {
	return domain.NewMemory(m.memoryRepository.NextID(), typ, who, time.Now(), what, where, m.getEmbedding(what))
}

func (m *MemoryFactory) getEmbedding(sentence string) *domain.Embedding {
	if sentence == "" {
		return nil
	}
	embedding, err := m.embedder.Embed(sentence)
	if err != nil {
		return nil
	}
	return &embedding
}
