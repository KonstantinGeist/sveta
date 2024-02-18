package web

type PageContentExtractor interface {
	ExtractPageContentFromURL(url string) (string, error)
}
