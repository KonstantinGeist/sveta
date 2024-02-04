package domain

import (
	"sort"
	"time"
)

type MemoryType int

const (
	// MemoryTypeDialog an utterance in the chat
	MemoryTypeDialog = MemoryType(iota)
	// MemoryTypeAction an action (for example, "I entered the chat")
	MemoryTypeAction
)

type Memory struct {
	ID        string
	Type      MemoryType
	Who       string
	When      time.Time
	What      string
	Where     string
	Embedding *Embedding // nullable
}

type MemoryFilter struct {
	Types        []MemoryType
	Who          string
	Where        string
	What         string
	LatestCount  int
	NotOlderThan *time.Time
}

type EmbeddingFilter struct {
	Types               []MemoryType
	Where               string
	Embeddings          []Embedding
	TopCount            int
	SurroundingCount    int
	ExcludedIDs         []string
	SimilarityThreshold float64
}

type MemoryRepository interface {
	NextID() string // should be time-sortable
	Store(memory *Memory) error
	Find(filter MemoryFilter) ([]*Memory, error)
	FindByEmbeddings(filter EmbeddingFilter) ([]*Memory, error)
	RemoveAll() error
}

type MemoryFactory interface {
	NewMemory(typ MemoryType, who string, what string, where string) *Memory
}

func NewMemory(id string, typ MemoryType, who string, when time.Time, what string, where string, embedding *Embedding) *Memory {
	return &Memory{
		ID:        id,
		Type:      typ,
		Who:       who,
		When:      when,
		What:      what,
		Where:     where,
		Embedding: embedding,
	}
}

func FilterMemoriesByTypes(memories []*Memory, types []MemoryType) []*Memory {
	result := make([]*Memory, 0, len(memories))
	for _, memory := range memories {
		if IsMemoryTypeInSlice(memory.Type, types) {
			result = append(result, memory)
		}
	}
	return result
}

func IsMemoryTypeInSlice(memoryType MemoryType, slice []MemoryType) bool {
	for _, s := range slice {
		if memoryType == s {
			return true
		}
	}
	return false
}

func MergeMemories(a []*Memory, b ...*Memory) []*Memory {
	c := make([]*Memory, 0, len(a)+len(b))
	c = append(c, a...)
	c = append(c, b...)
	result := make([]*Memory, 0, len(c))
	uniqueSet := make(map[string]struct{})
	for _, m := range c {
		if m == nil {
			continue
		}
		_, exists := uniqueSet[m.ID]
		if exists {
			continue
		}
		uniqueSet[m.ID] = struct{}{}
		result = append(result, m)
	}
	sort.SliceStable(result, func(i, j int) bool { // to preserve the order when two memories are created at the same time
		return result[i].When.Before(result[j].When)
	})
	return result
}

func GetMemoryIDs(memories []*Memory) []string {
	ids := make([]string, 0, len(memories))
	for _, memory := range memories {
		ids = append(ids, memory.ID)
	}
	return ids
}

func UniqueMemories(memories []*Memory) []*Memory {
	uniqueSet := make(map[string]struct{})
	result := make([]*Memory, 0, len(memories))
	for _, memory := range memories {
		_, exists := uniqueSet[memory.ID]
		if exists {
			continue
		}
		uniqueSet[memory.ID] = struct{}{}
		result = append(result, memory)
	}
	return result
}
