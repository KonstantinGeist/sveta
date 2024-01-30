package aifilters

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/mvdan/xurls"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO internationalize
const couldntLoadURLFormatMessage = "%s Description: \"no description because the URL failed to load\""
const urlDescriptionFormatMessage = "%s\nSome content from the link above: \"%s\""

type htmlFilter struct {
	logger         common.Logger
	maxContentSize int
}

// NewHTMLFilter this AI filter allows the AI agent to see the content of the given URLs.
// Limitations:
// - only sees the first URL, if there are several URLs in a message
// - the whole AI agent can crash if the given URL dynamically produces infinite output (see common.ReadAllFromURL)
func NewHTMLFilter(logger common.Logger, config *common.Config) domain.AIFilter {
	return &htmlFilter{
		logger:         logger,
		maxContentSize: config.GetIntOrDefault("htmlMaxPageContentSize", 1000),
	}
}

func (h *htmlFilter) Apply(who, what, where string, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	urls := xurls.Relaxed.FindAllString(what, -1)
	if len(urls) == 0 {
		return nextAIFilterFunc(who, what, where)
	}
	url := urls[0]                 // let's do it with only one URL so far (a known limitation)
	if common.IsImageFormat(url) { // for images, we have ImageFilter
		return nextAIFilterFunc(who, what, where)
	}
	pageContent, err := h.extractPageContentFromURL(url)
	if err != nil {
		// It's important to add `couldntLoadURLFormatMessage` so that the main LLM correctly respond that the URL doesn't load.
		return nextAIFilterFunc(who, fmt.Sprintf(couldntLoadURLFormatMessage, what), where)
	}
	pageContent = h.processPageContent(pageContent)
	if pageContent == "" {
		return nextAIFilterFunc(who, fmt.Sprintf(couldntLoadURLFormatMessage, what), where)
	}
	return nextAIFilterFunc(who, fmt.Sprintf(urlDescriptionFormatMessage, what, pageContent), where)
}

func (h *htmlFilter) extractPageContentFromURL(url string) (string, error) {
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

func (h *htmlFilter) processPageContent(pageContent string) string {
	if len(pageContent) > h.maxContentSize {
		pageContent = pageContent[0:h.maxContentSize]
	}
	pageContent = strings.ReplaceAll(pageContent, "\n", " ")
	pageContent = strings.ReplaceAll(pageContent, "\r", "")
	return strings.TrimSpace(pageContent)
}
