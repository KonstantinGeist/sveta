package bio

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type filter struct {
	aiContext        *domain.AIContext
	provider         Provider
	memoryRepository domain.MemoryRepository
	memoryFactory    domain.MemoryFactory
	logger           common.Logger
	loaded           map[string]bool // where => isLoaded
}

func NewFilter(
	aiContext *domain.AIContext,
	bioProvider Provider,
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		aiContext:        aiContext,
		provider:         bioProvider,
		memoryRepository: memoryRepository,
		memoryFactory:    memoryFactory,
		logger:           logger,
		loaded:           make(map[string]bool),
	}
}

func (f *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if !f.loaded[context.Where] {
		f.loadBioFacts(context.Where)
		f.loaded[context.Where] = true
	}
	return nextAIFilterFunc(context)
}

func (f *filter) loadBioFacts(where string) {
	bioFacts, err := f.provider.GetBioFacts()
	if err != nil {
		f.logger.Log("failed to load bio facts")
		return
	}
	for index, bioFact := range bioFacts {
		f.logger.Log(fmt.Sprintf("Loading bio fact #%d...\n", index))
		memory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, f.aiContext.AgentName+"Biography", bioFact, where)
		memory.When = time.Time{}
		err = f.memoryRepository.Store(memory)
		if err != nil {
			f.logger.Log("failed to store bio facts as memory")
			return
		}
	}
}
