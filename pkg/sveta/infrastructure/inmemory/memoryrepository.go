package inmemory

import (
	"sort"

	"github.com/google/uuid"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type memoryRepository struct {
	memories []*domain.Memory
}

func NewMemoryRepository() domain.MemoryRepository {
	return &memoryRepository{}
}

func (r *memoryRepository) NextID() string {
	return uuid.NewString()
}

func (r *memoryRepository) Store(memory *domain.Memory) error {
	r.memories = append(r.memories, memory)
	return nil
}

func (r *memoryRepository) Find(filter domain.MemoryFilter) ([]*domain.Memory, error) {
	if filter.LatestCount < 0 || filter.LatestCount > len(r.memories) {
		filter.LatestCount = len(r.memories)
	}
	var result []*domain.Memory
	count := 0
	for i := len(r.memories) - 1; i >= 0; i-- {
		entry := r.memories[i]
		if !memoryFilterApplies(filter, entry) {
			continue
		}
		result = append(result, entry) // NOTE: underlying memory objects are shared
		count++
		if count == filter.LatestCount {
			break
		}
	}
	// Reverses the slice, because we were adding to it backwards.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result, nil
}

func (r *memoryRepository) FindByEmbeddings(filter domain.EmbeddingFilter) ([]*domain.Memory, error) {
	var similarities []struct {
		Memory     *domain.Memory
		Index      int
		Similarity float64
	}
	for index, memory := range r.memories {
		if !embeddingFilterAppliesWithoutEmbedding(filter, memory) {
			continue
		}
		if memory.Embedding == nil {
			continue
		}
		similarity := memory.Embedding.GetBestSimilarityTo(filter.Embeddings)
		if similarity < filter.SimilarityThreshold { // ignore sentences which are too different
			continue
		}
		similarities = append(similarities, struct {
			Memory     *domain.Memory
			Index      int
			Similarity float64
		}{Memory: memory, Index: index, Similarity: similarity})
	}
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Similarity > similarities[j].Similarity
	})
	topCount := filter.TopCount
	if topCount > len(similarities) {
		topCount = len(similarities)
	}
	similarities = similarities[0:topCount]
	var result []*domain.Memory
	for _, similarity := range similarities {
		for index := 0; index < filter.SurroundingCount*2+1; index++ {
			finalIndex := similarity.Index - (filter.SurroundingCount*2+1)/2 + index
			if finalIndex < 0 {
				finalIndex = 0
			}
			if finalIndex >= len(r.memories) {
				finalIndex = len(r.memories) - 1
			}
			result = append(result, r.memories[finalIndex])
		}
	}
	return domain.UniqueMemories(result), nil
}

func (r *memoryRepository) RemoveAll() error {
	var newMems []*domain.Memory
	for _, mem := range r.memories {
		if mem.When.IsZero() {
			newMems = append(newMems, mem)
		}
	}
	r.memories = newMems
	return nil
}

func memoryFilterApplies(filter domain.MemoryFilter, memory *domain.Memory) bool {
	if len(filter.Types) > 0 && !domain.IsMemoryTypeInSlice(memory.Type, filter.Types) {
		return false
	}
	if filter.Who != "" && memory.Who != filter.Who {
		return false
	}
	if filter.Where != "" && memory.Where != filter.Where {
		return false
	}
	if filter.What != "" && memory.What != filter.What {
		return false
	}
	if filter.NotOlderThan != nil && memory.When.Before(*filter.NotOlderThan) {
		return false
	}
	return true
}

func embeddingFilterAppliesWithoutEmbedding(filter domain.EmbeddingFilter, memory *domain.Memory) bool {
	if len(filter.Types) > 0 && !domain.IsMemoryTypeInSlice(memory.Type, filter.Types) {
		return false
	}
	if filter.Where != "" && memory.Where != filter.Where {
		return false
	}
	if common.IsStringInSlice(memory.ID, filter.ExcludedIDs) {
		return false
	}
	return true
}
