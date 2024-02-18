package domain

const DataKeyInput = "input"
const DataKeyOutput = "output"

type PassContext struct {
	// Data a map of arbitrary values which can be passed from pass to pass.
	Data                map[string]any
	EnabledCapabilities []*Capability
}

func NewPassContext() *PassContext {
	return &PassContext{
		Data: make(map[string]any),
	}
}

func (a *PassContext) IsCapabilityEnabled(name string) bool {
	for _, capability := range a.EnabledCapabilities {
		if capability.Name == name {
			return true
		}
	}
	return false
}

func (a *PassContext) Memories(key string) []*Memory {
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

func (a *PassContext) Memory(key string) *Memory {
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

func (a *PassContext) MemoryCoalesced(keys []string) *Memory {
	for _, key := range keys {
		memory := a.Memory(key)
		if memory != nil {
			return memory
		}
	}
	return nil
}

func (a *PassContext) WithMemories(key string, memories []*Memory) *PassContext {
	a.Data[key] = memories
	return a
}

func (a *PassContext) WithMemory(key string, memory *Memory) *PassContext {
	a.Data[key] = memory
	return a
}

func (a *PassContext) WithCapabilities(capabilities []*Capability) *PassContext {
	a.EnabledCapabilities = capabilities
	return a
}
