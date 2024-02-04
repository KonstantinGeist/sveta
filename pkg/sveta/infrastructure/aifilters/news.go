package aifilters

import (
	"fmt"
	"strings"
	"time"

	"github.com/mmcdole/gofeed/rss"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type newsFilter struct {
	memoryRepository domain.MemoryRepository
	memoryFactory    domain.MemoryFactory
	logger           common.Logger
	newsLoaded       map[string]bool // where => isLoaded
	maxCount         int
	newsSourceURL    string
}

func NewNewsFilter(
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &newsFilter{
		memoryRepository: memoryRepository,
		memoryFactory:    memoryFactory,
		logger:           logger,
		newsLoaded:       make(map[string]bool),
		maxCount:         config.GetIntOrDefault("newsMaxCount", 100),
		newsSourceURL:    config.GetStringOrDefault("newsSourceURL", "http://www.independent.co.uk/rss"),
	}
}

func (n *newsFilter) Apply(who, what, where string, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if !n.newsLoaded[where] {
		n.loadNews(where)
		n.newsLoaded[where] = true
	}
	return nextAIFilterFunc(who, what, where)
}

func (n *newsFilter) loadNews(where string) {
	data, err := common.ReadAllFromURL(n.newsSourceURL)
	if err != nil {
		n.logger.Log("failed to load news from the URL")
		return
	}
	fp := rss.Parser{}
	rssFeed, _ := fp.Parse(strings.NewReader(string(data)))
	for index, item := range rssFeed.Items {
		line := fmt.Sprintf("Published Date: %s. Title: \"%s\". Description: \"%s\"", item.PubDate, strings.TrimSpace(item.Title), strings.TrimSpace(item.Description))
		_ = index
		fmt.Printf("Loading news #%d...\n", index)
		memory := n.memoryFactory.NewMemory(domain.MemoryTypeDialog, "News", line, where)
		memory.When = time.Time{}
		err = n.memoryRepository.Store(memory)
		if err != nil {
			n.logger.Log("failed to store news as memory")
			return
		}
		if index > n.maxCount {
			break
		}
	}
}
