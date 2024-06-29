package filesystem

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type memoryRepository struct {
	wrapped domain.MemoryRepository
	file    *os.File
	logger  common.Logger
	mutex   sync.Mutex
}

type jsonMemory struct {
	ID        string `json:"id"`
	Type      int    `json:"type"`
	Who       string `json:"who"`
	When      int64  `json:"when"`
	What      string `json:"what"`
	Where     string `json:"where"`
	Embedding string `json:"embedding"`
}

func NewMemoryRepository(
	wrapped domain.MemoryRepository,
	config *common.Config,
	logger common.Logger,
) domain.MemoryRepository {
	memoryFilePath := config.GetString("memoryFilePath")
	file, _ := os.OpenFile(memoryFilePath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	r := &memoryRepository{
		wrapped: wrapped,
		file:    file,
		logger:  logger,
	}
	r.rememberMemories(memoryFilePath)
	return r
}

func (m *memoryRepository) NextID() string {
	return m.wrapped.NextID()
}

func (m *memoryRepository) Store(memory *domain.Memory) error {
	err := m.wrapped.Store(memory)
	if err != nil {
		return err
	}
	if m.file == nil || memory.IsTransient {
		return nil
	}
	jsonMemory := jsonMemory{
		ID:        memory.ID,
		Type:      int(memory.Type),
		Who:       removeNewLines(memory.Who),
		When:      memory.When.UnixNano(),
		What:      removeNewLines(memory.What),
		Where:     removeNewLines(memory.Where),
		Embedding: memory.Embedding.ToFormattedValues(),
	}
	jsonMemoryBytes, err := json.Marshal(jsonMemory)
	if err != nil {
		return err
	}
	m.mutex.Lock()
	defer m.mutex.Unlock()
	_, err = m.file.WriteString(string(jsonMemoryBytes))
	if err != nil {
		return err
	}
	_, err = m.file.WriteString("\n")
	if err != nil {
		return err
	}
	return m.file.Sync()
}

func (m *memoryRepository) Find(filter domain.MemoryFilter) ([]*domain.Memory, error) {
	return m.wrapped.Find(filter)
}

func (m *memoryRepository) FindByEmbeddings(filter domain.EmbeddingFilter) ([]*domain.Memory, error) {
	return m.wrapped.FindByEmbeddings(filter)
}

func (m *memoryRepository) RemoveAll() error {
	return m.wrapped.RemoveAll()
}

func (m *memoryRepository) rememberMemories(memoryFilePath string) {
	lines, err := common.ReadAllLines(memoryFilePath)
	if err != nil {
		return
	}
	for _, line := range lines {
		var jsonMemory jsonMemory
		err := json.Unmarshal([]byte(line), &jsonMemory)
		if err != nil {
			m.logger.Log(fmt.Sprintf("failed to parse memory in the memory file: %s", line))
			continue
		}
		embedding, err := domain.NewEmbeddingFromFormattedValues(jsonMemory.Embedding)
		if err != nil {
			m.logger.Log(fmt.Sprintf("failed to parse memory in the memory file: %s", line))
			continue
		}
		memory := domain.NewMemory(
			jsonMemory.ID,
			domain.MemoryType(jsonMemory.Type),
			jsonMemory.Who,
			time.Unix(0, jsonMemory.When),
			jsonMemory.What,
			jsonMemory.Where,
			&embedding,
		)
		_ = m.wrapped.Store(memory)
	}
}

func removeNewLines(str string) string {
	return strings.ReplaceAll(str, "\n", "")
}
