package domain

type respondFunc func(who, what, where string) (string, error)

// AIService is the main orchestrator of the whole AI: it receives a list of AI filters and runs them one after another.
// Additionally, it has various functions for debugging/control: remove all memory, remember actions, change context etc.
type AIService struct {
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

func (a *AIService) Respond(who, what, where string) (string, error) {
	return a.applyAIFilterAtIndex(who, what, where, 0)
}

func (a *AIService) RememberAction(who, what, where string) error {
	memory := a.memoryFactory.NewMemory(MemoryTypeAction, who, what, where)
	return a.memoryRepository.Store(memory)
}

func (a *AIService) RememberDialog(who, what, where string) error {
	memory := a.memoryFactory.NewMemory(MemoryTypeDialog, who, what, where)
	return a.memoryRepository.Store(memory)
}

// ForgetEverything removes all memory. Useful for debugging.
func (a *AIService) ForgetEverything() error {
	return a.memoryRepository.RemoveAll()
}

// ChangeAgentDescription changes the context. Useful for debugging or when the persona of the AI should change.
func (a *AIService) ChangeAgentDescription(context string) error {
	a.aiContext.AgentDescription = context
	return nil
}

func (a *AIService) applyAIFilterAtIndex(who, what, where string, index int) (string, error) {
	var nextFilterFunc NextFilterFunc
	if index < len(a.aiFilters)-1 {
		nextFilterFunc = func(who, what, where string) (string, error) {
			return a.applyAIFilterAtIndex(who, what, where, index+1)
		}
	} else {
		nextFilterFunc = func(who, what, where string) (string, error) {
			return what, nil
		}
	}
	return a.aiFilters[index].Apply(who, what, where, nextFilterFunc)
}
