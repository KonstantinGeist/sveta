package news

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/workingmemory"
)

const newsCapabillity = "news"

type pass struct {
	provider          Provider
	memoryRepository  domain.MemoryRepository
	memoryFactory     domain.MemoryFactory
	summaryRepository domain.SummaryRepository
	logger            common.Logger
	loaded            map[string]bool // where => isLoaded
	maxNewsCount      int
}

func NewPass(
	newsProvider Provider,
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	summaryRepository domain.SummaryRepository,
	config *common.Config,
	logger common.Logger,
) domain.Pass {
	return &pass{
		provider:          newsProvider,
		memoryRepository:  memoryRepository,
		memoryFactory:     memoryFactory,
		summaryRepository: summaryRepository,
		logger:            logger,
		loaded:            make(map[string]bool),
		maxNewsCount:      config.GetIntOrDefault("newsMaxCount", 100),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        newsCapabillity,
			Description: "looks for the answer in the world news",
			IsMaskable:  true,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(newsCapabillity) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	if p.loaded[inputMemory.Where] {
		return nextPassFunc(context)
	}
	summary, err := p.summaryRepository.FindByWhere(inputMemory.Where)
	if err != nil {
		p.logger.Log("failed to find summary: " + err.Error())
		return nextPassFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	// Do not load news before there's at least 1 memory and a summary, otherwise
	// it can dominate the context and may end up ignoring the user's query.
	if len(workingMemories) < 1 || summary == nil {
		return nextPassFunc(context)
	}
	p.loadNews(inputMemory.Where)
	p.loaded[inputMemory.Where] = true
	return nextPassFunc(context)
}

func (p *pass) loadNews(where string) {
	newsItems, err := p.provider.GetNews(p.maxNewsCount)
	if err != nil {
		p.logger.Log("failed to load news")
		return
	}
	for index, newsItem := range newsItems {
		p.logger.Log(fmt.Sprintf("Loading news #%d...\n", index))
		line := fmt.Sprintf("Published Date: %s. Title: \"%s\". Description: \"%s\"", newsItem.PublishedDate, newsItem.Title, newsItem.Description)
		memory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, "News", line, where)
		memory.When = time.Time{}
		err = p.memoryRepository.Store(memory)
		if err != nil {
			p.logger.Log("failed to store news as memory")
			return
		}
	}
}
