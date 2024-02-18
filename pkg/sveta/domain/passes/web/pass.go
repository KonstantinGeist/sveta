package web

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const couldntLoadURLFormatMessage = "%s Description: \"no description because the URL failed to load\""
const urlDescriptionFormatMessage = "%s\nContext found at the URL: \"%s\"\nQuery: \"%s\" (answer using the provided context, but slightly reformulate it in the language of your persona)"

const webCapability = "web"

type pass struct {
	urlFinder            common.URLFinder
	pageContentExtractor PageContentExtractor
	logger               common.Logger
	maxContentSize       int
}

// NewPass this AI pass allows the AI agent to see the content of the given URLs.
// Limitations:
// - only sees the first URL, if there are several URLs in a message
// - the whole AI agent can crash if the given URL dynamically produces infinite output (see common.ReadAllFromURL)
func NewPass(
	urlFinder common.URLFinder,
	pageContentExtractor PageContentExtractor,
	config *common.Config,
	logger common.Logger,
) domain.Pass {
	return &pass{
		urlFinder:            urlFinder,
		pageContentExtractor: pageContentExtractor,
		logger:               logger,
		maxContentSize:       config.GetIntOrDefault("htmlMaxPageContentSize", 1000),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        webCapability,
			Description: "answers the user query by analyzing web pages (if URLs are provided)",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(webCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	urls := p.urlFinder.FindURLs(inputMemory.What)
	if len(urls) == 0 {
		return nextPassFunc(context)
	}
	url := urls[0]                 // let's do it with only one URL so far (a known limitation)
	if common.IsImageFormat(url) { // for images, we have a vision pass
		return nextPassFunc(context)
	}
	pageContent, err := p.pageContentExtractor.ExtractPageContentFromURL(url)
	if err != nil {
		// It's important to add `couldntLoadURLFormatMessage` so that the main LLM correctly respond that the URL doesn't load.
		inputMemory.What = fmt.Sprintf(couldntLoadURLFormatMessage, inputMemory.What)
		return nextPassFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
	}
	pageContent = p.preprocessPageContent(pageContent)
	if pageContent == "" {
		inputMemory.What = fmt.Sprintf(couldntLoadURLFormatMessage, inputMemory.What)
		return nextPassFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
	}
	whatWithoutURL := p.removeURL(inputMemory.What, url)
	inputMemory.What = fmt.Sprintf(urlDescriptionFormatMessage, url, pageContent, whatWithoutURL)
	return nextPassFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
}

func (p *pass) preprocessPageContent(pageContent string) string {
	if len(pageContent) > p.maxContentSize {
		pageContent = pageContent[0:p.maxContentSize]
	}
	pageContent = strings.ReplaceAll(pageContent, "\n", " ")
	pageContent = strings.ReplaceAll(pageContent, "\r", "")
	return strings.TrimSpace(pageContent)
}

func (p *pass) removeURL(what, url string) string {
	return strings.TrimSpace(strings.ReplaceAll(what, url, ""))
}
