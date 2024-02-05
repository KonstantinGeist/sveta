package web

import "github.com/mvdan/xurls"

type URLFinder struct{}

func NewURLFinder() *URLFinder {
	return &URLFinder{}
}

func (u *URLFinder) FindURLs(str string) []string {
	return xurls.Relaxed.FindAllString(str, -1)
}
