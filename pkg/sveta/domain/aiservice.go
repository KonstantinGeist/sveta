package domain

import (
	"sort"
	"sync"
)

type capability struct {
	Name      string
	IsEnabled bool
}

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
	capabilities      []capability
}

func NewAIService(
	memoryRepository MemoryRepository,
	memoryFactory MemoryFactory,
	summaryRepository SummaryRepository,
	functionService *FunctionService,
	aiContext *AIContext,
	aiFilters []AIFilter,
) *AIService {
	var capabilities []capability
	for _, filter := range aiFilters {
		for _, c := range filter.Capabilities() {
			capabilities = append(capabilities, capability{
				Name:      c.Name,
				IsEnabled: true,
			})
		}
	}
	return &AIService{
		memoryRepository:  memoryRepository,
		memoryFactory:     memoryFactory,
		summaryRepository: summaryRepository,
		functionService:   functionService,
		aiContext:         aiContext,
		aiFilters:         aiFilters,
		capabilities:      capabilities,
	}
}

// Respond see API.Respond
func (a *AIService) Respond(who, what, where string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	inputMemory := a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where)
	aiFilterContext := NewAIFilterContext().WithMemory(DataKeyInput, inputMemory)
	aiFilterContext.EnabledCapabilities = a.listEnabledCapabilities()
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

func (a *AIService) ListCapabilities() []string {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	var result []string
	for _, c := range a.capabilities {
		result = append(result, c.Name)
	}
	sort.Strings(result)
	return result
}

func (a *AIService) EnableCapability(name string, value bool) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	for i, c := range a.capabilities {
		if c.Name == name {
			a.capabilities[i].IsEnabled = value
			break
		}
	}
	return nil
}

func (a *AIService) listEnabledCapabilities() []string {
	var result []string
	for _, c := range a.capabilities {
		if c.IsEnabled {
			result = append(result, c.Name)
		}
	}
	sort.Strings(result)
	return result
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
