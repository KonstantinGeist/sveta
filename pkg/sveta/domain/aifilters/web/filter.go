package web

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO internationalize
const couldntLoadURLFormatMessage = "%s Description: \"no description because the URL failed to load\""
const urlDescriptionFormatMessage = "%s\nContext found at the URL: \"%s\"\nQuery: \"%s\" (answer using the provided context, but slightly reformulate it in the language of your persona)"

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

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	urls := f.urlFinder.FindURLs(inputMemory.What)
	if len(urls) == 0 {
		return nextAIFilterFunc(context)
	}
	url := urls[0]                 // let's do it with only one URL so far (a known limitation)
	if common.IsImageFormat(url) { // for images, we have ImageFilter
		return nextAIFilterFunc(context)
	}
	pageContent, err := f.pageContentExtractor.ExtractPageContentFromURL(url)
	if err != nil {
		// It's important to add `couldntLoadURLFormatMessage` so that the main LLM correctly respond that the URL doesn't load.
		inputMemory.What = fmt.Sprintf(couldntLoadURLFormatMessage, inputMemory.What)
		return nextAIFilterFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
	}
	pageContent = f.preprocessPageContent(pageContent)
	if pageContent == "" {
		inputMemory.What = fmt.Sprintf(couldntLoadURLFormatMessage, inputMemory.What)
		return nextAIFilterFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
	}
	whatWithoutURL := f.removeURL(inputMemory.What, url)
	inputMemory.What = fmt.Sprintf(urlDescriptionFormatMessage, url, pageContent, whatWithoutURL)
	return nextAIFilterFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
}

func (f *filter) preprocessPageContent(pageContent string) string {
	if len(pageContent) > f.maxContentSize {
		pageContent = pageContent[0:f.maxContentSize]
	}
	pageContent = strings.ReplaceAll(pageContent, "\n", " ")
	pageContent = strings.ReplaceAll(pageContent, "\r", "")
	return strings.TrimSpace(pageContent)
}

func (f *filter) removeURL(what, url string) string {
	return strings.TrimSpace(strings.ReplaceAll(what, url, ""))
}
