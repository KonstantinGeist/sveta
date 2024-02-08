package summary

import (
	"fmt"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type filter struct {
	aiContext           *domain.AIContext
	memoryRepository    domain.MemoryRepository
	summaryRepository   domain.SummaryRepository
	responseService     *domain.ResponseService
	logger              common.Logger
	workingMemorySize   int
	workingMemoryMaxAge time.Duration
}

func NewFilter(
	aiContext *domain.AIContext,
	memoryRepository domain.MemoryRepository,
	summaryRepository domain.SummaryRepository,
	responseService *domain.ResponseService,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		aiContext:           aiContext,
		memoryRepository:    memoryRepository,
		summaryRepository:   summaryRepository,
		responseService:     responseService,
		logger:              logger,
		workingMemorySize:   config.GetIntOrDefault(domain.ConfigKeyWorkingMemorySize, 5),
		workingMemoryMaxAge: config.GetDurationOrDefault(domain.ConfigKeyWorkingMemoryMaxAge, time.Hour),
	}
}

func (f *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	var output struct {
		Summary1 string `json:"summary1"`
		Summary2 string `json:"summary2"`
		Summary3 string `json:"summary3"`
	}
	workingMemories, err := f.recallFromWorkingMemory(context.Where)
	if err != nil {
		f.logger.Log("failed to summarize: " + err.Error())
		return nextAIFilterFunc(context)
	}
	summary, err := f.summaryRepository.FindByWhere(context.Where)
	if err != nil {
		f.logger.Log("failed to summarize: " + err.Error())
		return nextAIFilterFunc(context)
	}
	formattedForSummarizer := f.formatMemoriesForSummarizer(summary, workingMemories)
	err = f.getSummarizerResponseService().RespondToQueryWithJSON(
		fmt.Sprintf("%s\nSummarize the chat history above into 3 summaries maximum (if possible).", formattedForSummarizer),
		&output,
	)
	if err != nil {
		f.logger.Log("failed to summarize: " + err.Error())
		return nextAIFilterFunc(context)
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
	err = f.summaryRepository.Store(context.Where, strings.TrimSpace(strings.Join(summaries, " ")))
	if err != nil {
		f.logger.Log("failed to summarize: " + err.Error())
	}
	return nextAIFilterFunc(context)
}

func (f *filter) recallFromWorkingMemory(where string) ([]*domain.Memory, error) {
	// Note that we don't want to recall the latest entries if they're too old (they're most likely already irrelevant).
	notOlderThan := time.Now().Add(-f.workingMemoryMaxAge)
	return f.memoryRepository.Find(domain.MemoryFilter{
		Types:        []domain.MemoryType{domain.MemoryTypeDialog, domain.MemoryTypeAction},
		Where:        where,
		LatestCount:  f.workingMemorySize,
		NotOlderThan: &notOlderThan,
	})
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

func (f *filter) formatMemoriesForSummarizer(summary *string, memories []*domain.Memory) string {
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
