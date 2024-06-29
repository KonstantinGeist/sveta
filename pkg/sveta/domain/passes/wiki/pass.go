package wiki

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/rewrite"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

const wikiCapability = "wiki"

type pass struct {
	responseService                *domain.ResponseService
	memoryFactory                  domain.MemoryFactory
	memoryRepository               domain.MemoryRepository
	articleProvider                ArticleProvider
	wordFrequencyProvider          WordFrequencyProvider
	logger                         common.Logger
	maxArticleCount                int
	maxArticleSentenceCount        int
	wordSizeThreshold              int
	wordFrequencyPositionThreshold int
}

func NewPass(
	responseService *domain.ResponseService,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	articleProvider ArticleProvider,
	wordFrequencyProvider WordFrequencyProvider,
	config *common.Config,
	logger common.Logger,
) domain.Pass {
	return &pass{
		responseService:                responseService,
		memoryFactory:                  memoryFactory,
		memoryRepository:               memoryRepository,
		articleProvider:                articleProvider,
		wordFrequencyProvider:          wordFrequencyProvider,
		logger:                         logger,
		maxArticleCount:                config.GetIntOrDefault("wikiMaxArticleCount", 2),
		maxArticleSentenceCount:        config.GetIntOrDefault("wikiMaxArticleSentenceCount", 3),
		wordSizeThreshold:              config.GetIntOrDefault("wikiWordSizeThreshold", 2),
		wordFrequencyPositionThreshold: config.GetIntOrDefault("wikiWordFrequencyPositionThreshold", 4000),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        wikiCapability,
			Description: "looks for the answer on Wikipedia",
			IsMaskable:  true,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(wikiCapability) {
		return nextPassFunc(context)
	}
	inputMemoryForResponse := context.MemoryCoalesced([]string{rewrite.DataKeyRewrittenInput, domain.DataKeyInput})
	if inputMemoryForResponse == nil {
		return nextPassFunc(context)
	}
	intputMemoryForApply := context.Memory(domain.DataKeyInput)
	if intputMemoryForApply == nil {
		return nextPassFunc(context)
	}
	if !p.shouldApply(intputMemoryForApply.What) && !p.shouldApply(inputMemoryForResponse.What) {
		return nextPassFunc(context)
	}
	var output struct {
		ArticleName string `json:"articleName"`
	}
	err := p.getWikiResponseService().RespondToQueryWithJSON(p.formatQuery(inputMemoryForResponse.What), &output)
	if err != nil {
		p.logger.Log(err.Error())
		return nextPassFunc(context)
	}
	if output.ArticleName == "" {
		p.logger.Log("article name not found")
		return nextPassFunc(context)
	}
	output.ArticleName = p.fixArticleName(output.ArticleName)
	articleNames, err := p.articleProvider.Search(output.ArticleName, p.maxArticleCount)
	if err != nil {
		p.logger.Log(err.Error())
		return nextPassFunc(context)
	}
	for _, articleName := range articleNames {
		summary, err := p.articleProvider.GetSummary(articleName, p.maxArticleSentenceCount)
		if err != nil {
			p.logger.Log(err.Error())
			return nextPassFunc(context)
		}
		if summary == "" {
			continue
		}
		summary = "\"" + summary + "\""
		if !p.memoryExists(summary, inputMemoryForResponse.Where) {
			err = p.storeMemory(summary, inputMemoryForResponse.Where)
			if err != nil {
				p.logger.Log(err.Error())
				return nextPassFunc(context)
			}
		}
	}
	return nextPassFunc(context)
}

// shouldApply a heuristic to avoid looking for information in a Wikipedia article if the message is very trivial/banal,
// i.e. contains only most popular words
func (p *pass) shouldApply(what string) bool {
	what = strings.ToLower(what)
	what = strings.ReplaceAll(what, "\n", " ")
	what = strings.TrimSpace(nonAlphanumericRegex.ReplaceAllString(what, ""))
	split := strings.Split(what, " ")
	for _, word := range split {
		if utf8.RuneCountInString(word) < p.wordSizeThreshold {
			continue
		}
		position := p.wordFrequencyProvider.GetPosition(word)
		if position > p.wordFrequencyPositionThreshold || position == -1 {
			return true
		}
	}
	return false
}

func (p *pass) storeMemory(what, where string) error {
	memory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, "SearchResult", what, where)
	memory.When = time.Time{}
	memory.IsTransient = true
	return p.memoryRepository.Store(memory)
}

func (p *pass) getWikiResponseService() *domain.ResponseService {
	wikiAIContext := domain.NewAIContext("WikiLLM", "You're WikiLLM, an intelligent assistant which can find the best Wiki article for the given topic. You pay attention to the most important words/phrases.", "")
	return p.responseService.WithAIContext(wikiAIContext)
}

func (p *pass) memoryExists(what, where string) bool {
	memories, err := p.memoryRepository.Find(domain.MemoryFilter{
		Types:       []domain.MemoryType{domain.MemoryTypeDialog},
		Where:       where,
		What:        what,
		LatestCount: 1,
	})
	if err != nil {
		p.logger.Log(err.Error())
		return false
	}
	return len(memories) > 0
}

func (p *pass) formatQuery(what string) string {
	return fmt.Sprintf("In what Wikipedia article can we find information related to this sentence: \"%s\" ?", what)
}

func (p *pass) fixArticleName(articleName string) string {
	articleName = p.removeWikiURLPrefixIfAny(articleName)
	articleName = common.RemoveDoubleQuotesIfAny(articleName)
	return common.RemoveSingleQuotesIfAny(articleName)
}

func (p *pass) removeWikiURLPrefixIfAny(articleName string) string {
	// Sometimes, the model returns the URL of the article, instead of just the article name.
	const wikiURLPrefix = "https://en.wikipedia.org/wiki/"
	if strings.HasPrefix(articleName, wikiURLPrefix) {
		articleName = articleName[len(wikiURLPrefix):]
	}
	return articleName
}
