package news

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/workingmemory"
)

type filter struct {
	provider          Provider
	memoryRepository  domain.MemoryRepository
	memoryFactory     domain.MemoryFactory
	summaryRepository domain.SummaryRepository
	logger            common.Logger
	loaded            map[string]bool // where => isLoaded
	maxNewsCount      int
}

func NewFilter(
	newsProvider Provider,
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	summaryRepository domain.SummaryRepository,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		provider:          newsProvider,
		memoryRepository:  memoryRepository,
		memoryFactory:     memoryFactory,
		summaryRepository: summaryRepository,
		logger:            logger,
		loaded:            make(map[string]bool),
		maxNewsCount:      config.GetIntOrDefault("newsMaxCount", 100),
	}
}

func (f *filter) Capabilities() []domain.AIFilterCapability {
	return []domain.AIFilterCapability{
		{
			Name:        "news",
			Description: "retrieves the latest world news if an answer to the user query can potentially be found in the news",
			CanBeMasked: true,
		},
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	if f.loaded[inputMemory.Where] {
		return nextAIFilterFunc(context)
	}
	summary, err := f.summaryRepository.FindByWhere(inputMemory.Where)
	if err != nil {
		f.logger.Log("failed to find summary: " + err.Error())
		return nextAIFilterFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	// Do not load news before there's at least 1 memory and a summary, otherwise
	// it can dominate the context and may end up ignoring the user's query.
	if len(workingMemories) < 1 || summary == nil {
		return nextAIFilterFunc(context)
	}
	f.loadNews(inputMemory.Where)
	f.loaded[inputMemory.Where] = true
	return nextAIFilterFunc(context)
}

func (f *filter) loadNews(where string) {
	newsItems, err := f.provider.GetNews(f.maxNewsCount)
	if err != nil {
		f.logger.Log("failed to load news")
		return
	}
	for index, newsItem := range newsItems {
		f.logger.Log(fmt.Sprintf("Loading news #%d...\n", index))
		line := fmt.Sprintf("Published Date: %s. Title: \"%s\". Description: \"%s\"", newsItem.PublishedDate, newsItem.Title, newsItem.Description)
		memory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, "News", line, where)
		memory.When = time.Time{}
		err = f.memoryRepository.Store(memory)
		if err != nil {
			f.logger.Log("failed to store news as memory")
			return
		}
	}
}
