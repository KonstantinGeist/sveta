package web

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO internationalize
const couldntLoadURLFormatMessage = "%s Description: \"no description because the URL failed to load\""
const urlDescriptionFormatMessage = "%s\nSome content from the link above: \"%s\""

type filter struct {
	urlFinder            common.URLFinder
	pageContentExtractor PageContentExtractor
	logger               common.Logger
	maxContentSize       int
}

// NewFilter this AI filter allows the AI agent to see the content of the given URLs.
// Limitations:
// - only sees the first URL, if there are several URLs in a message
// - the whole AI agent can crash if the given URL dynamically produces infinite output (see common.ReadAllFromURL)
func NewFilter(
	urlFinder common.URLFinder,
	pageContentExtractor PageContentExtractor,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		urlFinder:            urlFinder,
		pageContentExtractor: pageContentExtractor,
		logger:               logger,
		maxContentSize:       config.GetIntOrDefault("htmlMaxPageContentSize", 1000),
	}
}

func (h *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	urls := h.urlFinder.FindURLs(context.What)
	if len(urls) == 0 {
		return nextAIFilterFunc(context)
	}
	url := urls[0]                 // let's do it with only one URL so far (a known limitation)
	if common.IsImageFormat(url) { // for images, we have ImageFilter
		return nextAIFilterFunc(context)
	}
	pageContent, err := h.pageContentExtractor.ExtractPageContentFromURL(url)
	if err != nil {
		// It's important to add `couldntLoadURLFormatMessage` so that the main LLM correctly respond that the URL doesn't load.
		return nextAIFilterFunc(context.WithWhat(fmt.Sprintf(couldntLoadURLFormatMessage, context.What)))
	}
	pageContent = h.processPageContent(pageContent)
	if pageContent == "" {
		return nextAIFilterFunc(context.WithWhat(fmt.Sprintf(couldntLoadURLFormatMessage, context.What)))
	}
	return nextAIFilterFunc(context.WithWhat(fmt.Sprintf(urlDescriptionFormatMessage, context.What, pageContent)))
}

func (h *filter) processPageContent(pageContent string) string {
	if len(pageContent) > h.maxContentSize {
		pageContent = pageContent[0:h.maxContentSize]
	}
	pageContent = strings.ReplaceAll(pageContent, "\n", " ")
	pageContent = strings.ReplaceAll(pageContent, "\r", "")
	return strings.TrimSpace(pageContent)
}
