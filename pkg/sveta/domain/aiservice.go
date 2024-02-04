package domain

import "sync"

type respondFunc func(who, what, where string) (string, error)

// AIService is the main orchestrator of the whole AI: it receives a list of AI filters and runs them one after another.
// Additionally, it has various functions for debugging/control: remove all memory, remember actions, change context etc.\
type AIService struct {
	mutex            sync.Mutex
	memoryRepository MemoryRepository
	memoryFactory    MemoryFactory
	aiContext        *AIContext
	aiFilters        []AIFilter
}

func NewAIService(
	memoryRepository MemoryRepository,
	memoryFactory MemoryFactory,
	aiContext *AIContext,
	aiFilters []AIFilter,
) *AIService {
	return &AIService{
		memoryRepository: memoryRepository,
		memoryFactory:    memoryFactory,
		aiContext:        aiContext,
		aiFilters:        aiFilters,
	}
}

// Respond see API.Respond
func (a *AIService) Respond(who, what, where string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.applyAIFilterAtIndex(who, what, where, 0)
}

// RememberAction see API.RememberAction
func (a *AIService) RememberAction(who, what, where string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	memory := a.memoryFactory.NewMemory(MemoryTypeAction, who, what, where)
	return a.memoryRepository.Store(memory)
}

// RememberDialog see API.RememberDialog
func (a *AIService) RememberDialog(who, what, where string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	memory := a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where)
	return a.memoryRepository.Store(memory)
}

// ClearAllMemory see API.ClearAllMemory
func (a *AIService) ClearAllMemory() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.memoryRepository.RemoveAll()
}

// ChangeAgentDescription see API.ChangeAgentDescription
func (a *AIService) ChangeAgentDescription(context string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.aiContext.AgentDescription = context
	return nil
}

func (a *AIService) applyAIFilterAtIndex(who, what, where string, index int) (string, error) {
	var nextAIFilterFunc NextAIFilterFunc
	if index < len(a.aiFilters)-1 {
		nextAIFilterFunc = func(who, what, where string) (string, error) {
			return a.applyAIFilterAtIndex(who, what, where, index+1)
		}
	} else {
		nextAIFilterFunc = func(who, what, where string) (string, error) {
			return what, nil
		}
	}
	return a.aiFilters[index].Apply(who, what, where, nextAIFilterFunc)
}
