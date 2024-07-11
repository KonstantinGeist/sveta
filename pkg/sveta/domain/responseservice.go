package domain

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
)

var ErrFailedToResponse = errors.New("failed to respond")

// ResponseService makes it possible to respond to memories with text, or JSON.
type ResponseService struct {
	aiContext             *AIContext
	languageModelSelector *LanguageModelSelector
	embedder              Embedder
	memoryFactory         MemoryFactory
	summaryRepository     SummaryRepository
	logger                common.Logger
	retryCount            int
	textTemperature       float64
	jsonTemperature       float64
}

func NewResponseService(
	aiContext *AIContext,
	languageModelSelector *LanguageModelSelector,
	embedder Embedder,
	memoryFactory MemoryFactory,
	summaryRepository SummaryRepository,
	config *common.Config,
	logger common.Logger,
) *ResponseService {
	return &ResponseService{
		aiContext:             aiContext,
		languageModelSelector: languageModelSelector,
		embedder:              embedder,
		memoryFactory:         memoryFactory,
		summaryRepository:     summaryRepository,
		logger:                logger,
		retryCount:            config.GetIntOrDefault(ConfigKeyResponseRetryCount, 3),
		textTemperature:       config.GetFloat(ConfigKeyResponseTextTemperature),
		jsonTemperature:       config.GetFloat(ConfigKeyResponseJSONTemperature),
	}
}

func (r *ResponseService) WithAIContext(aiContext *AIContext) *ResponseService {
	clone := *r
	clone.aiContext = aiContext
	return &clone
}

func (r *ResponseService) WithLanguageModelSelector(selector *LanguageModelSelector) *ResponseService {
	clone := *r
	clone.languageModelSelector = selector
	return &clone
}

// RespondToMemoriesWithText responds to the given list of memories as a large language model.
func (r *ResponseService) RespondToMemoriesWithText(memories []*Memory, responseMode ResponseMode) (string, error) {
	if len(memories) == 0 {
		return "", nil
	}
	dialogAndActionMemories := FilterMemoriesByTypes(memories, []MemoryType{MemoryTypeDialog})
	languageModel := r.languageModelSelector.Select(responseMode)
	announcedTime := time.Now()
	summary := r.getSummary(memories)
	dialogPrompt := languageModel.PromptFormatter().FormatPrompt(FormatOptions{
		AgentName:                r.aiContext.AgentName,
		AgentDescription:         r.aiContext.AgentDescription,
		AgentDescriptionReminder: r.aiContext.AgentDescriptionReminder,
		Summary:                  summary,
		AnnouncedTime:            &announcedTime,
		Memories:                 dialogAndActionMemories,
	})
	completeOptions := DefaultCompleteOptions
	if responseMode == ResponseModeNormal {
		completeOptions = completeOptions.WithTemperature(r.textTemperature)
	} else {
		completeOptions = completeOptions.WithTemperature(r.jsonTemperature) // the reranker must have a lower temperature, similar to JSON
	}
	return r.complete(
		dialogPrompt,
		completeOptions,
		memories,
		languageModel,
	)
}

// RespondToQueryWithJSON responds to the given query in the JSON format and automatically fills `obj`'s property.
func (r *ResponseService) RespondToQueryWithJSON(query string, jsonObject any) error {
	jsonOutputSchema, err := json.Marshal(jsonObject)
	if err != nil {
		return err
	}
	queryMemories := []*Memory{r.memoryFactory.NewMemory(MemoryTypeDialog, "User", query, "")}
	languageModel := r.languageModelSelector.Select(ResponseModeJSON)
	dialogPrompt := languageModel.PromptFormatter().FormatPrompt(FormatOptions{
		AgentName:                r.aiContext.AgentName,
		AgentDescription:         r.aiContext.AgentDescription,
		AgentDescriptionReminder: r.aiContext.AgentDescriptionReminder,
		Memories:                 queryMemories,
		JSONOutputSchema:         string(jsonOutputSchema),
	})
	response, err := r.complete(
		dialogPrompt,
		DefaultCompleteOptions.WithJSONMode(true).WithTemperature(r.jsonTemperature),
		queryMemories,
		languageModel,
	)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(response), jsonObject)
}

// For both RespondToMemoriesWithText(..) and RespondToQueryWithJSON(..)
func (r *ResponseService) complete(prompt string, completeOptions CompleteOptions, memories []*Memory, languageModel LanguageModel) (string, error) {
	if len(memories) == 0 {
		return "", ErrFailedToResponse
	}
	for i := 0; i < r.retryCount; i++ {
		response, err := languageModel.Complete(prompt, completeOptions)
		if err != nil {
			return "", err
		}
		cleanResponse := languageModel.ResponseCleaner().CleanResponse(CleanOptions{
			Prompt:    prompt,
			Response:  response,
			AgentName: r.aiContext.AgentName,
			Memories:  memories,
		})
		// Sometimes, a model can just repeat the user's name.
		if strings.ToLower(cleanResponse) == strings.ToLower(LastMemory(memories).Who) {
			continue
		}
		if cleanResponse != "" {
			return cleanResponse, nil
		}
	}
	return "", ErrFailedToResponse
}

func (r *ResponseService) getSummary(memories []*Memory) string {
	where := LastMemory(memories).Where
	summary, err := r.summaryRepository.FindByWhere(where)
	if err != nil {
		r.logger.Log(err.Error())
		return ""
	}
	if summary == nil {
		return ""
	}
	return *summary
}
