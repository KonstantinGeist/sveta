package wiki

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type filter struct {
	responseService         *domain.ResponseService
	memoryFactory           domain.MemoryFactory
	memoryRepository        domain.MemoryRepository
	articleProvider         ArticleProvider
	logger                  common.Logger
	maxArticleCount         int
	maxArticleSentenceCount int
	messageSizeThreshold    int
}

func NewFilter(
	responseService *domain.ResponseService,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	articleProvider ArticleProvider,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		responseService:         responseService,
		memoryFactory:           memoryFactory,
		memoryRepository:        memoryRepository,
		articleProvider:         articleProvider,
		logger:                  logger,
		maxArticleCount:         config.GetIntOrDefault("wikiMaxArticleCount", 2),
		maxArticleSentenceCount: config.GetIntOrDefault("wikiMaxArticleSentenceCount", 3),
		messageSizeThreshold:    config.GetIntOrDefault("wikiMessageSizeThreshold", 8),
	}
}

func (w *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if utf8.RuneCountInString(context.What) < w.messageSizeThreshold {
		return nextAIFilterFunc(context)
	}
	var output struct {
		ArticleName string `json:"articleName"`
	}
	err := w.getWikiResponseService().RespondToQueryWithJSON(w.formatQuery(context.What), &output)
	if err != nil {
		w.logger.Log(err.Error())
		return nextAIFilterFunc(context)
	}
	if output.ArticleName == "" {
		w.logger.Log("article name not found")
		return nextAIFilterFunc(context)
	}
	output.ArticleName = w.fixArticleName(output.ArticleName)
	articleNames, err := w.articleProvider.Search(output.ArticleName, w.maxArticleCount)
	if err != nil {
		w.logger.Log(err.Error())
		return nextAIFilterFunc(context)
	}
	for _, articleName := range articleNames {
		summary, err := w.articleProvider.GetSummary(articleName, w.maxArticleSentenceCount)
		if err != nil {
			w.logger.Log(err.Error())
			return nextAIFilterFunc(context)
		}
		if summary == "" {
			continue
		}
		summary = "\"" + summary + "\""
		if !w.memoryExists(summary, context.Where) {
			err = w.storeMemory(summary, context.Where)
			if err != nil {
				w.logger.Log(err.Error())
				return "", err
			}
		}
	}
	return nextAIFilterFunc(context)
}

func (w *filter) storeMemory(what, where string) error {
	memory := w.memoryFactory.NewMemory(domain.MemoryTypeDialog, "SearchResult", what, where)
	memory.When = time.Time{}
	return w.memoryRepository.Store(memory)
}

func (w *filter) getWikiResponseService() *domain.ResponseService {
	wikiAIContext := domain.NewAIContext("WikiLLM", "You're WikiLLM, an intelligent assistant which can find the best Wiki article for the given topic.", "")
	return w.responseService.WithAIContext(wikiAIContext)
}

func (w *filter) memoryExists(what, where string) bool {
	memories, err := w.memoryRepository.Find(domain.MemoryFilter{
		Types:       []domain.MemoryType{domain.MemoryTypeDialog},
		Where:       where,
		What:        what,
		LatestCount: 1,
	})
	if err != nil {
		w.logger.Log(err.Error())
		return false
	}
	return len(memories) > 0
}

func (w *filter) formatQuery(what string) string {
	// TODO internationalize
	return fmt.Sprintf("In what Wikipedia article can we find information related to this sentence: \"%s\" ?", what)
}

func (w *filter) fixArticleName(articleName string) string {
	articleName = w.removeWikiURLPrefixIfAny(articleName)
	articleName = w.removeDoubleQuotesIfAny(articleName)
	return w.removeSingleQuotesIfAny(articleName)
}

func (w *filter) removeWikiURLPrefixIfAny(articleName string) string {
	// Sometimes, the model returns the URL of the article, instead of just the article name.
	const wikiURLPrefix = "https://en.wikipedia.org/wiki/"
	if strings.HasPrefix(articleName, wikiURLPrefix) {
		articleName = articleName[len(wikiURLPrefix):]
	}
	return articleName
}

func (w *filter) removeSingleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "'Hello'"
	if len(articleName) > 2 && articleName[0] == '\'' && articleName[len(articleName)-1] == '\'' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}

func (w *filter) removeDoubleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "\"Hello\""
	if len(articleName) > 2 && articleName[0] == '"' && articleName[len(articleName)-1] == '"' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}
