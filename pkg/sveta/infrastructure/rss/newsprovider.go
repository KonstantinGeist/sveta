package rss

import (
	"strings"

	"github.com/mmcdole/gofeed/rss"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/news"
)

type NewsProvider struct {
	url string
}

func NewNewsProvider(url string) *NewsProvider {
	return &NewsProvider{
		url: url,
	}
}

func (n *NewsProvider) GetNews(maxNewsCount int) ([]news.Item, error) {
	data, err := common.ReadAllFromURL(n.url)
	if err != nil {
		return nil, err
	}
	fp := rss.Parser{}
	rssFeed, err := fp.Parse(strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	result := make([]news.Item, 0, len(rssFeed.Items))
	for index, item := range rssFeed.Items {
		if index >= maxNewsCount {
			break
		}
		result = append(result, news.Item{
			PublishedDate: item.PubDate,
			Title:         strings.TrimSpace(removeKnownTags(item.Title)),
			Description:   strings.TrimSpace(removeKnownTags(item.Description)),
		})
	}
	return result, nil
}

func removeKnownTags(str string) string {
	str = strings.ReplaceAll(str, "<p>", "")
	return strings.ReplaceAll(str, "</p>", "")
}
