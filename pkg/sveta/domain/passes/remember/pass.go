package remember

import (
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const rememberCapability = "remember"

type pass struct {
	memoryRepository domain.MemoryRepository
}

func NewPass(memoryRepository domain.MemoryRepository) domain.Pass {
	return &pass{
		memoryRepository: memoryRepository,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        rememberCapability,
			Description: "stores the newly formed memories of the conversation",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(rememberCapability) {
		return nextPassFunc(context)
	}
	memories := []*domain.Memory{context.Memory(domain.DataKeyInput), context.Memory(domain.DataKeyOutput)}
	for _, memory := range memories {
		if memory != nil {
			err := p.memoryRepository.Store(memory)
			if err != nil {
				return err
			}
		}
	}
	return nextPassFunc(context)
}
