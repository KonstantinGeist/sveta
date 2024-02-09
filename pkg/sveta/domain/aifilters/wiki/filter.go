package wiki

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

type filter struct {
	responseService                *domain.ResponseService
	memoryFactory                  domain.MemoryFactory
	memoryRepository               domain.MemoryRepository
	articleProvider                ArticleProvider
	wordFrequencyProvider          WordFrequencyProvider
	logger                         common.Logger
	maxArticleCount                int
	maxArticleSentenceCount        int
	wordSizeThreshold              int
	wordFrequencyPositionThreshold int
}

func NewFilter(
	responseService *domain.ResponseService,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	articleProvider ArticleProvider,
	wordFrequencyProvider WordFrequencyProvider,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		responseService:                responseService,
		memoryFactory:                  memoryFactory,
		memoryRepository:               memoryRepository,
		articleProvider:                articleProvider,
		wordFrequencyProvider:          wordFrequencyProvider,
		logger:                         logger,
		maxArticleCount:                config.GetIntOrDefault("wikiMaxArticleCount", 2),
		maxArticleSentenceCount:        config.GetIntOrDefault("wikiMaxArticleSentenceCount", 3),
		wordSizeThreshold:              config.GetIntOrDefault("wikiWordSizeThreshold", 2),
		wordFrequencyPositionThreshold: config.GetIntOrDefault("wikiWordFrequencyPositionThreshold", 5000),
	}
}

func (f *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	if !f.shouldApply(context.What) {
		return nextAIFilterFunc(context)
	}
	var output struct {
		ArticleName string `json:"articleName"`
	}
	err := f.getWikiResponseService().RespondToQueryWithJSON(f.formatQuery(context.What), &output)
	if err != nil {
		f.logger.Log(err.Error())
		return nextAIFilterFunc(context)
	}
	if output.ArticleName == "" {
		f.logger.Log("article name not found")
		return nextAIFilterFunc(context)
	}
	output.ArticleName = f.fixArticleName(output.ArticleName)
	articleNames, err := f.articleProvider.Search(output.ArticleName, f.maxArticleCount)
	if err != nil {
		f.logger.Log(err.Error())
		return nextAIFilterFunc(context)
	}
	for _, articleName := range articleNames {
		summary, err := f.articleProvider.GetSummary(articleName, f.maxArticleSentenceCount)
		if err != nil {
			f.logger.Log(err.Error())
			return nextAIFilterFunc(context)
		}
		if summary == "" {
			continue
		}
		summary = "\"" + summary + "\""
		if !f.memoryExists(summary, context.Where) {
			err = f.storeMemory(summary, context.Where)
			if err != nil {
				f.logger.Log(err.Error())
				return "", err
			}
		}
	}
	return nextAIFilterFunc(context)
}

func (f *filter) shouldApply(what string) bool {
	what = strings.ToLower(what)
	what = strings.ReplaceAll(what, "\n", " ")
	what = strings.TrimSpace(nonAlphanumericRegex.ReplaceAllString(what, ""))
	split := strings.Split(what, " ")
	for _, word := range split {
		if utf8.RuneCountInString(word) < f.wordSizeThreshold {
			continue
		}
		position := f.wordFrequencyProvider.GetPosition(word)
		if position > f.wordFrequencyPositionThreshold {
			return true
		}
	}
	return false
}

func (f *filter) storeMemory(what, where string) error {
	memory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, "SearchResult", what, where)
	memory.When = time.Time{}
	return f.memoryRepository.Store(memory)
}

func (f *filter) getWikiResponseService() *domain.ResponseService {
	wikiAIContext := domain.NewAIContext("WikiLLM", "You're WikiLLM, an intelligent assistant which can find the best Wiki article for the given topic.", "")
	return f.responseService.WithAIContext(wikiAIContext)
}

func (f *filter) memoryExists(what, where string) bool {
	memories, err := f.memoryRepository.Find(domain.MemoryFilter{
		Types:       []domain.MemoryType{domain.MemoryTypeDialog},
		Where:       where,
		What:        what,
		LatestCount: 1,
	})
	if err != nil {
		f.logger.Log(err.Error())
		return false
	}
	return len(memories) > 0
}

func (f *filter) formatQuery(what string) string {
	// TODO internationalize
	return fmt.Sprintf("In what Wikipedia article can we find information related to this sentence: \"%s\" ?", what)
}

func (f *filter) fixArticleName(articleName string) string {
	articleName = f.removeWikiURLPrefixIfAny(articleName)
	articleName = f.removeDoubleQuotesIfAny(articleName)
	return f.removeSingleQuotesIfAny(articleName)
}

func (f *filter) removeWikiURLPrefixIfAny(articleName string) string {
	// Sometimes, the model returns the URL of the article, instead of just the article name.
	const wikiURLPrefix = "https://en.wikipedia.org/wiki/"
	if strings.HasPrefix(articleName, wikiURLPrefix) {
		articleName = articleName[len(wikiURLPrefix):]
	}
	return articleName
}

func (f *filter) removeSingleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "'Hello'"
	if len(articleName) > 2 && articleName[0] == '\'' && articleName[len(articleName)-1] == '\'' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}

func (f *filter) removeDoubleQuotesIfAny(articleName string) string {
	// Sometimes, the model returns the article name as "\"Hello\""
	if len(articleName) > 2 && articleName[0] == '"' && articleName[len(articleName)-1] == '"' {
		articleName = articleName[1 : len(articleName)-2]
	}
	return articleName
}
