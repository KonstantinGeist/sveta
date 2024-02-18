package workingmemory

import (
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const DataKeyWorkingMemory = "workingMemory"

const workingMemoryCapability = "workingMemory"

type filter struct {
	memoryRepository    domain.MemoryRepository
	memoryFactory       domain.MemoryFactory
	logger              common.Logger
	workingMemorySize   int
	workingMemoryMaxAge time.Duration
}

// NewFilter creates a filter which finds memories from the so-called "working memory" -- it's simply N latest memories
// (depends on  `workingMemorySize` specified in the config). Working memory is the basis for building proper dialog
// contexts (so that AI could hold continuous dialogs).
func NewFilter(
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		memoryRepository:    memoryRepository,
		memoryFactory:       memoryFactory,
		logger:              logger,
		workingMemorySize:   config.GetIntOrDefault(domain.ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge: config.GetDurationOrDefault(domain.ConfigKeyWorkingMemoryMaxAge, time.Hour),
	}
}

func (f *filter) Capabilities() []domain.AIFilterCapability {
	return []domain.AIFilterCapability{
		{
			Name:        workingMemoryCapability,
			Description: "retrieves the working memory",
		},
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	if !context.IsCapabilityEnabled(workingMemoryCapability) {
		return nextAIFilterFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	notOlderThan := time.Now().Add(-f.workingMemoryMaxAge)
	memories, err := f.memoryRepository.Find(domain.MemoryFilter{
		Types:        []domain.MemoryType{domain.MemoryTypeDialog},
		Where:        inputMemory.Where,
		LatestCount:  f.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
	if err != nil {
		f.logger.Log("failed to recall working memory: " + err.Error())
		return nextAIFilterFunc(context)
	}
	return nextAIFilterFunc(context.WithMemories(DataKeyWorkingMemory, memories))
}
