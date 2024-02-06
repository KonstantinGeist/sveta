package news

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type filter struct {
	provider         Provider
	memoryRepository domain.MemoryRepository
	memoryFactory    domain.MemoryFactory
	logger           common.Logger
	loaded           map[string]bool // where => isLoaded
	maxNewsCount     int
}

func NewFilter(
	newsProvider Provider,
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		provider:         newsProvider,
		memoryRepository: memoryRepository,
		memoryFactory:    memoryFactory,
		logger:           logger,
		loaded:           make(map[string]bool),
		maxNewsCount:     config.GetIntOrDefault("newsMaxCount", 100),
	}
}

func (f *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if !f.loaded[context.Where] {
		f.loadNews(context.Where)
		f.loaded[context.Where] = true
	}
	return nextAIFilterFunc(context)
}

func (f *filter) loadNews(where string) {
	newsItems, err := f.provider.GetNews(f.maxNewsCount)
	if err != nil {
		f.logger.Log("failed to load news")
		return
	}
	for index, newsItem := range newsItems {
		line := fmt.Sprintf("Published Date: %s. Title: \"%s\". Description: \"%s\"", newsItem.PublishedDate, newsItem.Title, newsItem.Description)
		f.logger.Log(fmt.Sprintf("Loading news #%d...\n", index))
		memory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, "News", line, where)
		memory.When = time.Time{}
		err = f.memoryRepository.Store(memory)
		if err != nil {
			f.logger.Log("failed to store news as memory")
			return
		}
	}
}
