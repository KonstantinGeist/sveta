package wiki

type ArticleProvider interface {
	Search(searchString string, maxArticleCount int) ([]string, error)
	GetSummary(articleName string, maxArticleSentenceCount int) (string, error)
}
