package summary

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/workingmemory"
)

type filter struct {
	aiContext             *domain.AIContext
	summaryRepository     domain.SummaryRepository
	responseService       *domain.ResponseService
	languageModelJobQueue *common.JobQueue
	logger                common.Logger
}

func NewFilter(
	aiContext *domain.AIContext,
	summaryRepository domain.SummaryRepository,
	responseService *domain.ResponseService,
	languageModelJobQueue *common.JobQueue,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		aiContext:             aiContext,
		summaryRepository:     summaryRepository,
		responseService:       responseService,
		languageModelJobQueue: languageModelJobQueue,
		logger:                logger,
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	inputMemory := context.Memory(domain.DataKeyInput)
	outputMemory := context.Memory(domain.DataKeyOutput)
	if inputMemory == nil || outputMemory == nil {
		return nextAIFilterFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	summary, err := f.summaryRepository.FindByWhere(inputMemory.Where)
	if err != nil {
		f.logger.Log("failed to summarize: " + err.Error())
		return nextAIFilterFunc(context)
	}
	formattedMemories := f.formatMemories(summary, domain.MergeMemories(workingMemories, []*domain.Memory{inputMemory, outputMemory}...))
	f.languageModelJobQueue.Enqueue(func() error {
		var output struct {
			Summary1              string `json:"summary1"`
			Summary2              string `json:"summary2"`
			Summary3              string `json:"summary3"`
			Summary4              string `json:"summary4"`
			Summary5              string `json:"summary5"`
			OpinionOnPeopleInChat string `json:"opinionOnPeopleInChat"`
		}
		err = f.getSummarizerResponseService().RespondToQueryWithJSON(
			fmt.Sprintf("%s\nSummarize the chat history above into 5 short summaries at most (if possible). Additionally, provide your your opinion on people in the chat using only adjectives. Example: \"User is friendly.\".", formattedMemories),
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
		finalSummary := strings.TrimSpace(strings.Join(summaries, " "))
		if output.OpinionOnPeopleInChat != "" {
			finalSummary += fmt.Sprintf("\n%s's opinion on people in the chat: \"%s\".", f.aiContext.AgentName, output.OpinionOnPeopleInChat)
		}
		return f.summaryRepository.Store(inputMemory.Where, finalSummary)
	})
	return nextAIFilterFunc(context)
}

func (f *filter) getSummarizerResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext(
		"SummaryLLM",
		"You're SummaryLLM, an intelligent assistant that summarizes the provided chat history by also taking the previous summary in consideration, if it exists. "+
			"When summarizing, pick the most relevant topics. "+
			"Example: \"User wants to meet up tomorrow.\", etc.",
		"",
	)
	return f.responseService.WithAIContext(rankerAIContext)
}

func (f *filter) formatMemories(summary *string, memories []*domain.Memory) string {
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
