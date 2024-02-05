package image

type URLFinder interface {
	FindURLs(str string) []string
}
