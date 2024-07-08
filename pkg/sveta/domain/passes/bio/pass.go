package bio

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const bioCapabillity = "bio"

type pass struct {
	aiContext        *domain.AIContext
	provider         Provider
	memoryRepository domain.MemoryRepository
	memoryFactory    domain.MemoryFactory
	logger           common.Logger
	loaded           map[string]bool // where => isLoaded
}

func NewPass(
	aiContext *domain.AIContext,
	bioProvider Provider,
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:        aiContext,
		provider:         bioProvider,
		memoryRepository: memoryRepository,
		memoryFactory:    memoryFactory,
		logger:           logger,
		loaded:           make(map[string]bool),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        bioCapabillity,
			Description: "looks for the answer in the biography",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(bioCapabillity) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	if !p.loaded[inputMemory.Where] {
		p.loadBioFacts(inputMemory.Where)
		p.loaded[inputMemory.Where] = true
	}
	return nextPassFunc(context)
}

func (p *pass) loadBioFacts(where string) {
	bioFacts, err := p.provider.GetBioFacts()
	if err != nil {
		p.logger.Log("failed to load bio facts")
		return
	}
	for index, bioFact := range bioFacts {
		p.logger.Log(fmt.Sprintf("Loading bio fact #%d...\n", index))
		memory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, bioFact, where)
		memory.When = time.Time{}
		memory.IsTransient = true
		err = p.memoryRepository.Store(memory)
		if err != nil {
			p.logger.Log("failed to store bio facts as memory")
			return
		}
	}
}
