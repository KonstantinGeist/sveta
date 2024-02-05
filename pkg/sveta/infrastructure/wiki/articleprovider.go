package wiki

import (
	gowiki "github.com/trietmn/go-wiki"

	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/wiki"
)

type articleProvider struct{}

func NewArticleProvider() wiki.ArticleProvider {
	return &articleProvider{}
}

func (a *articleProvider) Search(searchString string, maxArticleCount int) ([]string, error) {
	articleNames, _, err := gowiki.Search(searchString, maxArticleCount, true)
	return articleNames, err
}

func (a *articleProvider) GetSummary(articleName string, maxArticleSentenceCount int) (string, error) {
	return gowiki.Summary(articleName, maxArticleSentenceCount, -1, false, true)
}
