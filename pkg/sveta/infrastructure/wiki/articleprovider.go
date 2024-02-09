package wiki

import (
	"sync"

	gowiki "github.com/trietmn/go-wiki"
)

type ArticleProvider struct {
	mutex        sync.Mutex
	searchCache  map[string][]string
	summaryCache map[string]string
}

func NewArticleProvider() *ArticleProvider {
	return &ArticleProvider{
		searchCache:  make(map[string][]string),
		summaryCache: make(map[string]string),
	}
}

func (a *ArticleProvider) Search(searchString string, maxArticleCount int) ([]string, error) {
	cachedArticleNames, ok := a.searchInCache(searchString)
	if ok {
		return cachedArticleNames, nil
	}
	articleNames, _, err := gowiki.Search(searchString, maxArticleCount, true)
	if err != nil {
		return nil, err
	}
	a.cacheSearch(searchString, articleNames)
	return articleNames, err
}

func (a *ArticleProvider) GetSummary(articleName string, maxArticleSentenceCount int) (string, error) {
	cachedSummary, ok := a.getSummaryInCache(articleName)
	if ok {
		return cachedSummary, nil
	}
	summary, err := gowiki.Summary(articleName, maxArticleSentenceCount, -1, false, true)
	if err != nil {
		return "", err
	}
	a.cacheSummary(articleName, summary)
	return summary, err
}

func (a *ArticleProvider) searchInCache(searchString string) ([]string, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	articleNames, ok := a.searchCache[searchString]
	return articleNames, ok
}

func (a *ArticleProvider) getSummaryInCache(articleName string) (string, bool) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	summary, ok := a.summaryCache[articleName]
	return summary, ok
}

func (a *ArticleProvider) cacheSearch(searchString string, articleNames []string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.searchCache[searchString] = articleNames
}

func (a *ArticleProvider) cacheSummary(articleName, summary string) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.summaryCache[articleName] = summary
}
