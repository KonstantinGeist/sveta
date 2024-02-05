package wiki

import (
	gowiki "github.com/trietmn/go-wiki"
)

type ArticleProvider struct{}

func NewArticleProvider() *ArticleProvider {
	return &ArticleProvider{}
}

func (a *ArticleProvider) Search(searchString string, maxArticleCount int) ([]string, error) {
	articleNames, _, err := gowiki.Search(searchString, maxArticleCount, true)
	return articleNames, err
}

func (a *ArticleProvider) GetSummary(articleName string, maxArticleSentenceCount int) (string, error) {
	return gowiki.Summary(articleName, maxArticleSentenceCount, -1, false, true)
}
