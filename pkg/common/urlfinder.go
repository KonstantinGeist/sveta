package common

type URLFinder interface {
	FindURLs(str string) []string
}
