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

func (n *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if !n.loaded[context.Where] {
		n.loadNews(context.Where)
		n.loaded[context.Where] = true
	}
	return nextAIFilterFunc(context)
}

func (n *filter) loadNews(where string) {
	newsItems, err := n.provider.GetNews(n.maxNewsCount)
	if err != nil {
		n.logger.Log("failed to load news")
		return
	}
	for index, newsItem := range newsItems {
		line := fmt.Sprintf("Published Date: %s. Title: \"%s\". Description: \"%s\"", newsItem.PublishedDate, newsItem.Title, newsItem.Description)
		n.logger.Log(fmt.Sprintf("Loading news #%d...\n", index))
		memory := n.memoryFactory.NewMemory(domain.MemoryTypeDialog, "News", line, where)
		memory.When = time.Time{}
		err = n.memoryRepository.Store(memory)
		if err != nil {
			n.logger.Log("failed to store news as memory")
			return
		}
	}
}
