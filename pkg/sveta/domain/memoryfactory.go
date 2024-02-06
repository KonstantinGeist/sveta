package domain

type MemoryFactory interface {
	NewMemory(typ MemoryType, who string, what string, where string) *Memory
}
