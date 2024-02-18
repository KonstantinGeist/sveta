package news

type Provider interface {
	GetNews(maxNewsCount int) ([]Item, error)
}
