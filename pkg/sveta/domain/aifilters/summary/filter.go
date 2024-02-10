package summary

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/memory"
)

type filter struct {
	summaryRepository     domain.SummaryRepository
	responseService       *domain.ResponseService
	languageModelJobQueue *common.JobQueue
	logger                common.Logger
}

func NewFilter(
	summaryRepository domain.SummaryRepository,
	responseService *domain.ResponseService,
	languageModelJobQueue *common.JobQueue,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
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
	workingMemories := context.Memories(memory.DataKeyWorkingMemory)
	summary, err := f.summaryRepository.FindByWhere(inputMemory.Where)
	if err != nil {
		f.logger.Log("failed to summarize: " + err.Error())
		return nextAIFilterFunc(context)
	}
	formattedMemories := f.formatMemories(summary, domain.MergeMemories(workingMemories, []*domain.Memory{inputMemory, outputMemory}...))
	f.languageModelJobQueue.Enqueue(func() error {
		var output struct {
			Summary1 string `json:"summary1"`
			Summary2 string `json:"summary2"`
			Summary3 string `json:"summary3"`
		}
		err = f.getSummarizerResponseService().RespondToQueryWithJSON(
			fmt.Sprintf("%s\nSummarize the chat history above into 3 short summaries at most (if possible).", formattedMemories),
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
		return f.summaryRepository.Store(inputMemory.Where, strings.TrimSpace(strings.Join(summaries, " ")))
	})
	return nextAIFilterFunc(context)
}

func (f *filter) getSummarizerResponseService() *domain.ResponseService {
	// TODO internationalize
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
