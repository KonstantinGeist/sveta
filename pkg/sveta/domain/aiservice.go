package domain

import (
	"errors"
	"sort"
	"sync"
)

var errUnknownCapability = errors.New("unknown capability")

// AIService is the main orchestrator of the whole AI: it receives a list of passes and runs them one after another.
// Additionally, it has various functions for debugging/control: remove all memory, remember actions, change context etc.\
type AIService struct {
	mutex               sync.Mutex
	memoryRepository    MemoryRepository
	memoryFactory       MemoryFactory
	summaryRepository   SummaryRepository
	functionService     *FunctionService
	aiContext           *AIContext
	passes              []Pass
	capabilities        map[string]*Capability
	enabledCapabilities map[string]bool
}

func NewAIService(
	memoryRepository MemoryRepository,
	memoryFactory MemoryFactory,
	summaryRepository SummaryRepository,
	functionService *FunctionService,
	aiContext *AIContext,
	passes []Pass,
) *AIService {
	capabilities := make(map[string]*Capability)
	enabledCapabilities := make(map[string]bool)
	return &AIService{
		memoryRepository:    memoryRepository,
		memoryFactory:       memoryFactory,
		summaryRepository:   summaryRepository,
		functionService:     functionService,
		aiContext:           aiContext,
		passes:              passes,
		capabilities:        capabilities,
		enabledCapabilities: enabledCapabilities,
	}
}

// Respond see API.Respond
func (a *AIService) Respond(who, what, where string) (string, error) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	// Done lazily because some capabilities can be created dynamically after RegisterFunction(..)
	a.lazyLoadCapabilities()
	inputMemory := a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where)
	passContext := NewPassContext().WithMemory(DataKeyInput, inputMemory)
	passContext.EnabledCapabilities = a.listEnabledCapabilities()
	err := a.applyPassAtIndex(passContext, 0)
	if err != nil {
		return "", err
	}
	outputMemory := passContext.Memory(DataKeyOutput)
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
	for name, value := range a.enabledCapabilities {
		if !value {
			continue
		}
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}

func (a *AIService) EnableCapability(name string, value bool) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	_, ok := a.capabilities[name]
	if !ok {
		return errUnknownCapability
	}
	a.enabledCapabilities[name] = value
	return nil
}

func (a *AIService) lazyLoadCapabilities() {
	if len(a.capabilities) > 0 {
		return
	}
	for _, pass := range a.passes {
		for _, c := range pass.Capabilities() {
			a.enabledCapabilities[c.Name] = true
			a.capabilities[c.Name] = c
		}
	}
}

func (a *AIService) listEnabledCapabilities() []*Capability {
	var result []*Capability
	for name, value := range a.enabledCapabilities {
		if !value {
			continue
		}
		result = append(result, a.capabilities[name])
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func (a *AIService) applyPassAtIndex(context *PassContext, index int) error {
	var nextPassFunc NextPassFunc
	if index < len(a.passes)-1 {
		nextPassFunc = func(context *PassContext) error {
			return a.applyPassAtIndex(context, index+1)
		}
	} else {
		nextPassFunc = func(context *PassContext) error {
			return nil
		}
	}
	return a.passes[index].Apply(context, nextPassFunc)
}
