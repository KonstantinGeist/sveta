package domain

import "sync"

type respondFunc func(who, what, where string) (string, error)

// AIService is the main orchestrator of the whole AI: it receives a list of AI filters and runs them one after another.
// Additionally, it has various functions for debugging/control: remove all memory, remember actions, change context etc.\
type AIService struct {
	mutex             sync.Mutex
	memoryRepository  MemoryRepository
	memoryFactory     MemoryFactory
	summaryRepository SummaryRepository
	aiContext         *AIContext
	aiFilters         []AIFilter
}

func NewAIService(
	memoryRepository MemoryRepository,
	memoryFactory MemoryFactory,
	summaryRepository SummaryRepository,
	aiContext *AIContext,
	aiFilters []AIFilter,
) *AIService {
	return &AIService{
		memoryRepository:  memoryRepository,
		memoryFactory:     memoryFactory,
		summaryRepository: summaryRepository,
		aiContext:         aiContext,
		aiFilters:         aiFilters,
	}
}

// Respond see API.Respond
func (a *AIService) Respond(who, what, where string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.applyAIFilterAtIndex(NewAIFilterContext(who, what, where), 0)
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

func (a *AIService) GetSummary(where string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	summary, err := a.summaryRepository.FindByWhere(where)
	if err != nil {
		return "", err
	}
	if summary == nil {
		return "", nil
	}
	return *summary, nil
}

func (a *AIService) applyAIFilterAtIndex(context AIFilterContext, index int) (string, error) {
	var nextAIFilterFunc NextAIFilterFunc
	if index < len(a.aiFilters)-1 {
		nextAIFilterFunc = func(context AIFilterContext) (string, error) {
			return a.applyAIFilterAtIndex(context, index+1)
		}
	} else {
		nextAIFilterFunc = func(context AIFilterContext) (string, error) {
			return context.What, nil
		}
	}
	return a.aiFilters[index].Apply(context, nextAIFilterFunc)
}
