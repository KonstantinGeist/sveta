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

// RespondToMemoriesWithText responds to the given list of memories as a large language model.
func (r *ResponseService) RespondToMemoriesWithText(memories []*Memory, responseMode ResponseMode) (string, error) {
	if len(memories) == 0 {
		return "", nil
	}
	dialogAndActionMemories := FilterMemoriesByTypes(memories, []MemoryType{MemoryTypeDialog})
	languageModel := r.languageModelSelector.Select(memories, responseMode)
	announcedTime := time.Now()
	summary := r.getSummary(memories)
	dialogPrompt := languageModel.PromptFormatter2().FormatPrompt(FormatOptions{
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
	languageModel := r.languageModelSelector.Select(queryMemories, ResponseModeJSON)
	dialogPrompt := languageModel.PromptFormatter2().FormatPrompt(FormatOptions{
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
		dialogParticipants := r.collectDialogParticipants(memories)
		cleanResponse := r.cleanResponse(languageModel, response, dialogParticipants)
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

// Sometimes, the model can generate too much (for example, trying to complete other participants' dialogs), so we trim it.
func (r *ResponseService) cleanResponse(languageModel LanguageModel, response string, participants []string) string {
	promptFormatter := languageModel.PromptFormatter()
	agentNamePrefix := getAgentNameWithDelimiter(r.aiContext.AgentName, promptFormatter)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, agentNamePrefix) {
		response = response[len(agentNamePrefix):]
	}
	for _, participant := range participants {
		participantPrefix := getAgentNameWithDelimiter(participant, promptFormatter)
		foundIndex := strings.Index(response, participantPrefix)
		if foundIndex > 0 { // in the middle/at the end, when it wants to keep generating the dialog
			response = response[0:foundIndex]
		} else if foundIndex == 0 { // in the beginning, like: "User: "
			response = response[len(participantPrefix):]
		}
	}
	return strings.TrimSpace(response)
}

// For cleanResponse(..)
func (r *ResponseService) collectDialogParticipants(memories []*Memory) []string {
	resultSet := make(map[string]struct{})
	resultSet[r.aiContext.AgentName] = struct{}{}
	for _, memory := range memories {
		if (memory.Type == MemoryTypeDialog) && memory.Who != "" {
			resultSet[memory.Who] = struct{}{}
		}
	}
	participants := make([]string, 0, len(resultSet))
	for participant := range resultSet {
		participants = append(participants, participant)
	}
	return participants
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

// For cleanResponse(..)
func getAgentNameWithDelimiter(agentName string, promptFormatter PromptFormatter) string {
	memories := []*Memory{NewMemory("", MemoryTypeDialog, agentName, time.Now(), "", "", nil)}
	result := strings.TrimSpace(promptFormatter.FormatDialog(memories))
	return result
}
