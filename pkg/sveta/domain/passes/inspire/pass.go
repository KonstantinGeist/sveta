package inspire

import (
	"fmt"
	"math/rand"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const inspireCapability = "inspire"

const keywordCount = 3

type pass struct {
	aiContext             *domain.AIContext
	memoryFactory         domain.MemoryFactory
	responseService       *domain.ResponseService
	wordFrequencyProvider WordFrequencyProvider
	logger                common.Logger
}

func NewPass(
	aiContext *domain.AIContext,
	memoryFactory domain.MemoryFactory,
	responseService *domain.ResponseService,
	wordFrequencyProvider WordFrequencyProvider,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:             aiContext,
		memoryFactory:         memoryFactory,
		responseService:       responseService,
		wordFrequencyProvider: wordFrequencyProvider,
		logger:                logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        inspireCapability,
			Description: "creates a random inspiration quote similar to inspirabot",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(inspireCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	input := inputMemory.What
	if input != "inspire" {
		return nextPassFunc(context)
	}
	randomWords := p.getRandomWords()
	if len(randomWords) == 0 {
		p.logger.Log("failed to inspire: random words not found")
		return nextPassFunc(context)
	}
	p.logger.Log(fmt.Sprintf("INSPIRATIONAL keywords: %s\n", strings.Join(randomWords, ", ")))
	query := fmt.Sprintf("Create a demotivational quote and nothing else, based on the following keywords: %s. Output only the inspirational quote. The quote should be short, to put on a motivation poster. The quote must be thought-provoking.", strings.Join(randomWords, ", "))
	queryMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, "User", query, "")
	quote, err := p.getInspireResponseService().RespondToMemoriesWithText([]*domain.Memory{queryMemory}, domain.ResponseModeNormal)
	if err != nil {
		p.logger.Log("failed to inspire: " + err.Error())
		return nextPassFunc(context)
	}
	outputMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, quote, inputMemory.Where)
	context.Data[domain.DataKeyOutput] = outputMemory
	return nil
}

func (p *pass) getRandomWords() []string {
	var words []string
	for i := 0; i < keywordCount; i++ {
		randomPosition := rand.Intn(p.wordFrequencyProvider.MaxPosition())
		word := p.wordFrequencyProvider.GetWordAtPosition(randomPosition)
		if word == "" {
			continue
		}
		words = append(words, word)
	}
	return words
}

func (p *pass) getInspireResponseService() *domain.ResponseService {
	inspireAIContext := domain.NewAIContext("InspireLLM", "You are InspireLLM, an intelligent LLM which creates unique, demotivational quote and nothing else.", "")
	return p.responseService.WithAIContext(inspireAIContext)
}
