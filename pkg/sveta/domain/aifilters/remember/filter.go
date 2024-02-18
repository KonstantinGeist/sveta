package remember

import (
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const rememberCapability = "remember"

type filter struct {
	memoryRepository domain.MemoryRepository
}

func NewFilter(memoryRepository domain.MemoryRepository) domain.AIFilter {
	return &filter{
		memoryRepository: memoryRepository,
	}
}

func (f *filter) Capabilities() []domain.AIFilterCapability {
	return []domain.AIFilterCapability{
		{
			Name:        rememberCapability,
			Description: "stores the newly formed memories of the conversation",
		},
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	if !context.IsCapabilityEnabled(rememberCapability) {
		return nextAIFilterFunc(context)
	}
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
