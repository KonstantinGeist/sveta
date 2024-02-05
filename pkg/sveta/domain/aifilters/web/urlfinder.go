package web

type URLFinder interface {
	FindURLs(str string) []string
}
