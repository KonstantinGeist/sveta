package summary

import (
	"fmt"
	"math/rand"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/workingmemory"
)

const summaryCapability = "summary"

const maxSummaryCount = 3
const maxRandomWordCount = 1

type WordFrequencyProvider interface {
	MaxPosition() int
	GetWordAtPosition(position int) string
}

type pass struct {
	aiContext             *domain.AIContext
	summaryRepository     domain.SummaryRepository
	responseService       *domain.ResponseService
	wordFrequencyProvider WordFrequencyProvider
	languageModelJobQueue *common.JobQueue
	logger                common.Logger
}

func NewPass(
	aiContext *domain.AIContext,
	summaryRepository domain.SummaryRepository,
	responseService *domain.ResponseService,
	wordFrequencyProvider WordFrequencyProvider,
	languageModelJobQueue *common.JobQueue,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:             aiContext,
		summaryRepository:     summaryRepository,
		responseService:       responseService,
		wordFrequencyProvider: wordFrequencyProvider,
		languageModelJobQueue: languageModelJobQueue,
		logger:                logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        summaryCapability,
			Description: "summarizes the current conversation to have a better understanding of a long conversation",
		},
	}
}

func (p *pass) getRandomWords() []string {
	var words []string
	for i := 0; i < maxRandomWordCount; i++ {
		randomPosition := rand.Intn(p.wordFrequencyProvider.MaxPosition())
		word := p.wordFrequencyProvider.GetWordAtPosition(randomPosition)
		if word == "" {
			continue
		}
		words = append(words, word)
	}
	return words
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(summaryCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	outputMemory := context.Memory(domain.DataKeyOutput)
	if inputMemory == nil || outputMemory == nil {
		return nextPassFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	summary, err := p.summaryRepository.FindByWhere(inputMemory.Where)
	if err != nil {
		p.logger.Log("failed to summarize: " + err.Error())
		return nextPassFunc(context)
	}
	formattedMemories := p.formatMemories(summary, domain.MergeMemories(workingMemories, []*domain.Memory{inputMemory, outputMemory}...))
	p.languageModelJobQueue.Enqueue(func() error {
		var output struct {
			Summary1              string `json:"summary1"`
			Summary2              string `json:"summary2"`
			Summary3              string `json:"summary3"`
			Summary4              string `json:"summary4"`
			Summary5              string `json:"summary5"`
			OpinionOnPeopleInChat string `json:"opinionOnPeopleInChat"`
		}
		err = p.getSummarizerResponseService().RespondToQueryWithJSON(
			fmt.Sprintf("%s\nSummarize the chat history above into 5 short summaries at most (if possible). Additionally, provide your opinion on people in the chat using only adjectives. Example: \"User is friendly.\".", formattedMemories),
			&output,
		)
		if err != nil {
			return err
		}
		var summaries []string
		if output.Summary1 != "" {
			summaries = append(summaries, output.Summary1)
		}
		if output.Summary2 != "" {
			summaries = append(summaries, output.Summary2)
		}
		if output.Summary3 != "" {
			summaries = append(summaries, output.Summary3)
		}
		if output.Summary4 != "" {
			summaries = append(summaries, output.Summary4)
		}
		if output.Summary5 != "" {
			summaries = append(summaries, output.Summary5)
		}
		summaries = randomElements(summaries, maxSummaryCount)
		finalSummary := strings.TrimSpace(strings.Join(summaries, " "))
		randomWords := p.getRandomWords()
		finalSummary += fmt.Sprintf("\nPossible topic for discussion: \"%s\"\n", strings.Join(randomWords, " "))
		if output.OpinionOnPeopleInChat != "" {
			finalSummary += fmt.Sprintf("\n%s's opinion on people in the chat: \"%s\".", p.aiContext.AgentName, output.OpinionOnPeopleInChat)
		}
		return p.summaryRepository.Store(inputMemory.Where, finalSummary)
	})
	return nextPassFunc(context)
}

func randomElements(slice []string, n int) []string {
	if n > len(slice) {
		n = len(slice)
	}
	selected := make(map[int]bool)
	result := make([]string, 0, n)
	for len(result) < n {
		index := rand.Intn(len(slice))
		if !selected[index] {
			selected[index] = true
			result = append(result, slice[index])
		}
	}
	return result
}

func (p *pass) getSummarizerResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext(
		"SummaryLLM",
		"You're SummaryLLM, an intelligent assistant that summarizes the provided chat history by also taking the previous summary in consideration, if it exists. "+
			"When summarizing, pick the most relevant topics. "+
			"Example: \"User wants to meet up tomorrow.\", etc.",
		"",
	)
	return p.responseService.WithAIContext(rankerAIContext)
}

func (p *pass) formatMemories(summary *string, memories []*domain.Memory) string {
	var buf strings.Builder
	buf.WriteString("Chat history: ```\n")
	for _, memory := range memories {
		buf.WriteString(fmt.Sprintf("%s: %s\n\n", memory.Who, memory.What))
	}
	buf.WriteString("```\n\n")
	if summary != nil && *summary != "" {
		buf.WriteString(fmt.Sprintf("Previous summary: \"%s\"\n", *summary))
	}
	return buf.String()
}
