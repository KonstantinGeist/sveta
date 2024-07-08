package workingmemory

import (
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const DataKeyWorkingMemory = "workingMemory"

const workingMemoryCapability = "workingMemory"

type pass struct {
	memoryRepository    domain.MemoryRepository
	memoryFactory       domain.MemoryFactory
	logger              common.Logger
	workingMemorySize   int
	workingMemoryMaxAge time.Duration
}

// NewPass creates a pass which finds memories from the so-called "working memory" -- it's simply N latest memories
// (depends on  `workingMemorySize` specified in the config). Working memory is the basis for building proper dialog
// contexts (so that AI could hold continuous dialogs).
func NewPass(
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	config *common.Config,
	logger common.Logger,
) domain.Pass {
	return &pass{
		memoryRepository:    memoryRepository,
		memoryFactory:       memoryFactory,
		logger:              logger,
		workingMemorySize:   config.GetIntOrDefault(domain.ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge: config.GetDurationOrDefault(domain.ConfigKeyWorkingMemoryMaxAge, time.Hour),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        workingMemoryCapability,
			Description: "retrieves the working memory",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(workingMemoryCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	notOlderThan := time.Now().Add(-p.workingMemoryMaxAge)
	memories, err := p.memoryRepository.Find(domain.MemoryFilter{
		Types:        []domain.MemoryType{domain.MemoryTypeDialog},
		Where:        inputMemory.Where,
		LatestCount:  p.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
	if err != nil {
		p.logger.Log("failed to recall working memory: " + err.Error())
		return nextPassFunc(context)
	}
	return nextPassFunc(context.WithMemories(DataKeyWorkingMemory, memories))
}
