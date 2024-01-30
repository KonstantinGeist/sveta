package aifilters

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/http"

	"github.com/PuerkitoBio/goquery"
	"github.com/mvdan/xurls"
)

// TODO internationalize
const couldntLoadURLFormatMessage = "%s Description: \"no description because the URL failed to load\""
const urlDescriptionFormatMessage = "%s\nSome content from the link above: \"%s\""

type HTMLFilter struct {
	logger common.Logger
}

func NewHTMLFilter(logger common.Logger) domain.AIFilter {
	return &HTMLFilter{
		logger: logger,
	}
}

func (h *HTMLFilter) Apply(who, what, where string, nextFilterFunc domain.NextFilterFunc) (string, error) {
	urls := xurls.Relaxed.FindAllString(what, -1)
	if len(urls) == 0 {
		return nextFilterFunc(who, what, where)
	}
	url := urls[0] // let's do it with only one URL so far
	// TODO
	if strings.HasSuffix(url, ".jpg") || !strings.HasSuffix(url, ".jpeg") || strings.HasSuffix(url, ".png") {
		return nextFilterFunc(who, what, where)
	}
	page, err := http.ReadAllFromURL(url)
	if err != nil {
		h.logger.Log(err.Error())
		return nextFilterFunc(who, fmt.Sprintf(couldntLoadURLFormatMessage, what), where)
	}
	reader, err := goquery.NewDocumentFromReader(strings.NewReader(page))
	if err != nil {
		h.logger.Log(err.Error())
		return nextFilterFunc(who, fmt.Sprintf(couldntLoadURLFormatMessage, what), where)
	}
	plain := reader.Find("title").Text()
	found := reader.Find("p").Map(func(i int, selection *goquery.Selection) string {
		return selection.Text()
	})
	plain += " " + strings.Join(found, " ")
	if len(plain) > 1000 {
		plain = plain[0:1000]
	}
	plain = strings.ReplaceAll(plain, "\n", " ")
	plain = strings.ReplaceAll(plain, "\r", "")
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return nextFilterFunc(who, fmt.Sprintf(couldntLoadURLFormatMessage, what), where)
	}
	return nextFilterFunc(who, fmt.Sprintf(urlDescriptionFormatMessage, what, plain), where)
}
