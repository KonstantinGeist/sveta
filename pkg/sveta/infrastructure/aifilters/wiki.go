package aifilters

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	gowiki "github.com/trietmn/go-wiki"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type wikiFilter struct {
	responseService         *domain.ResponseService
	memoryFactory           domain.MemoryFactory
	memoryRepository        domain.MemoryRepository
	logger                  common.Logger
	maxArticleCount         int
	maxArticleSentenceCount int
	messageSizeThreshold    int
}

func NewWikiFilter(
	responseService *domain.ResponseService,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	logger common.Logger,
	config *common.Config,
) domain.AIFilter {
	return &wikiFilter{
		responseService:         responseService,
		memoryFactory:           memoryFactory,
		memoryRepository:        memoryRepository,
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
	err := w.responseService.RespondToQueryWithJSON(w.formatQuery(what), &output)
	if err != nil {
		w.logger.Log(err.Error())
		return nextAIFilterFunc(who, what, where)
	}
	if output.ArticleName == "" {
		w.logger.Log("article name not found")
		return nextAIFilterFunc(who, what, where)
	}
	output.ArticleName = w.removeWikiURLPrefixIfAny(output.ArticleName)
	output.ArticleName = w.removeSingleQuotesIfAny(output.ArticleName)
	search_result, _, err := gowiki.Search(output.ArticleName, w.maxArticleCount, true)
	if err != nil {
		w.logger.Log(err.Error())
		return nextAIFilterFunc(who, what, where)
	}
	for _, result := range search_result {
		summary, err := gowiki.Summary(result, w.maxArticleSentenceCount, -1, false, true)
		if err != nil {
			w.logger.Log(err.Error())
			return nextAIFilterFunc(who, what, where)
		}
		if summary == "" {
			continue
		}
		memory := w.memoryFactory.NewMemory(domain.MemoryTypeDialog, "SearchResult", summary, where)
		if err != nil {
			return "", err
		}
		memory.When = time.Time{}
		err = w.memoryRepository.Store(memory)
		if err != nil {
			return "", err
		}
	}
	return nextAIFilterFunc(who, what, where)
}

func (w *wikiFilter) formatQuery(what string) string {
	// TODO internationalize
	return fmt.Sprintf("In what Wikipedia article can we find information related to this sentence: \"%s\" ?", what)
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
