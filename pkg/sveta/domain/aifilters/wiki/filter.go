package wiki

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type wikiFilter struct {
	responseService         *domain.ResponseService
	memoryFactory           domain.MemoryFactory
	memoryRepository        domain.MemoryRepository
	articleProvider         ArticleProvider
	logger                  common.Logger
	maxArticleCount         int
	maxArticleSentenceCount int
	messageSizeThreshold    int
}

func NewWikiFilter(
	responseService *domain.ResponseService,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	articleProvider ArticleProvider,
	logger common.Logger,
	config *common.Config,
) domain.AIFilter {
	return &wikiFilter{
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

func (w *wikiFilter) Apply(who, what, where string, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if utf8.RuneCountInString(what) < w.messageSizeThreshold {
		return nextAIFilterFunc(who, what, where)
	}
	var output struct {
		Reasoning   string `json:"reasoning"`
		ArticleName string `json:"articleName"`
	}
	err := w.getWikiResponseService().RespondToQueryWithJSON(w.formatQuery(what), &output)
	if err != nil {
		w.logger.Log(err.Error())
		return nextAIFilterFunc(who, what, where)
	}
	if output.ArticleName == "" {
		w.logger.Log("article name not found")
		return nextAIFilterFunc(who, what, where)
	}
	output.ArticleName = w.fixArticleName(output.ArticleName)
	articleNames, err := w.articleProvider.Search(output.ArticleName, w.maxArticleCount)
	if err != nil {
		w.logger.Log(err.Error())
		return nextAIFilterFunc(who, what, where)
	}
	for _, articleName := range articleNames {
		summary, err := w.articleProvider.GetSummary(articleName, w.maxArticleSentenceCount)
		if err != nil {
			w.logger.Log(err.Error())
			return nextAIFilterFunc(who, what, where)
		}
		if summary == "" {
			continue
		}
		summary = "\"" + summary + "\""
		if !w.memoryExists(summary, where) {
			err = w.storeMemory(summary, where)
			if err != nil {
				w.logger.Log(err.Error())
				return "", err
			}
		}
	}
	return nextAIFilterFunc(who, what, where)
}

func (w *wikiFilter) storeMemory(what, where string) error {
	memory := w.memoryFactory.NewMemory(domain.MemoryTypeDialog, "SearchResult", what, where)
	memory.When = time.Time{}
	return w.memoryRepository.Store(memory)
}

func (w *wikiFilter) getWikiResponseService() *domain.ResponseService {
	wikiAIContext := domain.NewAIContext("WikiLLM", "You're WikiLLM, an intelligent assistant which can find the best Wiki article for the given topic.")
	return w.responseService.WithAIContext(wikiAIContext)
}

func (w *wikiFilter) memoryExists(what, where string) bool {
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

func (w *wikiFilter) formatQuery(what string) string {
	// TODO internationalize
	return fmt.Sprintf("In what Wikipedia article can we find information related to this sentence: \"%s\" ?", what)
}

func (w *wikiFilter) fixArticleName(articleName string) string {
	articleName = w.removeWikiURLPrefixIfAny(articleName)
	articleName = w.removeDoubleQuotesIfAny(articleName)
	return w.removeSingleQuotesIfAny(articleName)
}

func (w *wikiFilter) removeWikiURLPrefixIfAny(articleName string) string {
	// Sometimes, the model returns the URL of the article, instead of just the article name.
	const wikiURLPrefix = "https://en.wikipedia.org/wiki/"
	if strings.HasPrefix(articleName, wikiURLPrefix) {
		articleName = articleName[len(wikiURLPrefix):]
	}
	return articleName
}

func (w *wikiFilter) removeSingleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "'Hello'"
	if len(articleName) > 2 && articleName[0] == '\'' && articleName[len(articleName)-1] == '\'' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}

func (w *wikiFilter) removeDoubleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "\"Hello\""
	if len(articleName) > 2 && articleName[0] == '"' && articleName[len(articleName)-1] == '"' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}