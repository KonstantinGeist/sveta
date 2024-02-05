package web

import (
	"strings"

	"github.com/PuerkitoBio/goquery"

	"kgeyst.com/sveta/pkg/common"
)

type PageContentExtractor struct{}

func NewPageContentExtractor() *PageContentExtractor {
	return &PageContentExtractor{}
}

func (p *PageContentExtractor) ExtractPageContentFromURL(url string) (string, error) {
	// TODO add timeout
	page, err := common.ReadAllFromURL(url)
	if err != nil {
		return "", err
	}
	reader, err := goquery.NewDocumentFromReader(strings.NewReader(string(page)))
	if err != nil {
		return "", err
	}
	pageContenxt := reader.Find("title").Text()
	found := reader.Find("p").Map(func(i int, selection *goquery.Selection) string {
		return selection.Text()
	})
	pageContenxt += " " + strings.Join(found, " ")
	return pageContenxt, nil
}
