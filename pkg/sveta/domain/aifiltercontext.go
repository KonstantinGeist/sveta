package domain

const DataKeyInput = "input"
const DataKeyOutput = "output"

type AIFilterContext struct {
	// Data a map of arbitrary values which can be passed from filter to filter.
	Data map[string]any
}

func NewAIFilterContext() *AIFilterContext {
	return &AIFilterContext{
		Data: make(map[string]any),
	}
}

func (a *AIFilterContext) Memories(key string) []*Memory {
	value, ok := a.Data[key]
	if !ok {
		return nil
	}
	memories, ok := value.([]*Memory)
	if !ok {
		return nil
	}
	return memories
}

func (a *AIFilterContext) Memory(key string) *Memory {
	value, ok := a.Data[key]
	if !ok {
		return nil
	}
	memory, ok := value.(*Memory)
	if !ok {
		return nil
	}
	return memory
}

func (a *AIFilterContext) MemoryCoalesced(keys []string) *Memory {
	for _, key := range keys {
		memory := a.Memory(key)
		if memory != nil {
			return memory
		}
	}
	return nil
}

func (a *AIFilterContext) WithMemories(key string, memories []*Memory) *AIFilterContext {
	a.Data[key] = memories
	return a
}

func (a *AIFilterContext) WithMemory(key string, memory *Memory) *AIFilterContext {
	a.Data[key] = memory
	return a
}
