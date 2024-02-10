package remember

import (
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type filter struct {
	memoryRepository domain.MemoryRepository
}

func NewFilter(memoryRepository domain.MemoryRepository) domain.AIFilter {
	return &filter{
		memoryRepository: memoryRepository,
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	memories := []*domain.Memory{context.Memory(domain.DataKeyInput), context.Memory(domain.DataKeyOutput)}
	for _, memory := range memories {
		if memory != nil {
			err := f.memoryRepository.Store(memory)
			if err != nil {
				return err
			}
		}
	}
	return nextAIFilterFunc(context)
}
