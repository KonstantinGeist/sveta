package domain

import (
	"errors"
	"sync"
)

var errFailedToResponse = errors.New("failed to respond")

// AIService is the main orchestrator of the whole AI: it receives a list of AI filters and runs them one after another.
// Additionally, it has various functions for debugging/control: remove all memory, remember actions, change context etc.\
type AIService struct {
	mutex             sync.Mutex
	memoryRepository  MemoryRepository
	memoryFactory     MemoryFactory
	summaryRepository SummaryRepository
	functionService   *FunctionService
	aiContext         *AIContext
	aiFilters         []AIFilter
}

func NewAIService(
	memoryRepository MemoryRepository,
	memoryFactory MemoryFactory,
	summaryRepository SummaryRepository,
	functionService *FunctionService,
	aiContext *AIContext,
	aiFilters []AIFilter,
) *AIService {
	return &AIService{
		memoryRepository:  memoryRepository,
		memoryFactory:     memoryFactory,
		summaryRepository: summaryRepository,
		functionService:   functionService,
		aiContext:         aiContext,
		aiFilters:         aiFilters,
	}
}

// Respond see API.Respond
func (a *AIService) Respond(who, what, where string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	inputMemory := a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where)
	aiFilterContext := NewAIFilterContext().WithMemory(DataKeyInput, inputMemory)
	err := a.applyAIFilterAtIndex(aiFilterContext, 0)
	if err != nil {
		return "", err
	}
	outputMemory := aiFilterContext.Memory(DataKeyOutput)
	if outputMemory == nil {
		return "", nil
	}
	return outputMemory.What, nil
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
	err := a.memoryRepository.RemoveAll()
	if err != nil {
		return err
	}
	return a.summaryRepository.RemoveAll()
}

// ChangeAgentDescription see API.ChangeAgentDescription
func (a *AIService) ChangeAgentDescription(description string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.aiContext.AgentDescription = description
	return nil
}

func (a *AIService) ChangeAgentName(name string) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.aiContext.AgentName = name
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

func (a *AIService) RegisterFunction(functionDesc FunctionDesc) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	return a.functionService.RegisterFunction(functionDesc)
}

func (a *AIService) applyAIFilterAtIndex(context *AIFilterContext, index int) error {
	var nextAIFilterFunc NextAIFilterFunc
	if index < len(a.aiFilters)-1 {
		nextAIFilterFunc = func(context *AIFilterContext) error {
			return a.applyAIFilterAtIndex(context, index+1)
		}
	} else {
		nextAIFilterFunc = func(context *AIFilterContext) error {
			return nil
		}
	}
	return a.aiFilters[index].Apply(context, nextAIFilterFunc)
}
