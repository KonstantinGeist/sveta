package domain

import (
	"encoding/json"
	"errors"
	"fmt"
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
	config *common.Config,
	logger common.Logger,
) *ResponseService {
	return &ResponseService{
		aiContext:             aiContext,
		languageModelSelector: languageModelSelector,
		embedder:              embedder,
		memoryFactory:         memoryFactory,
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
	dialogAndActionMemories := FilterMemoriesByTypes(memories, []MemoryType{MemoryTypeDialog, MemoryTypeAction})
	languageModel := r.languageModelSelector.Select(memories, responseMode)
	promptFormatter := languageModel.PromptFormatter()
	promptEndMemories := r.generatePromptEndMemories()
	memoriesAsString := promptFormatter.FormatDialog(MergeMemories(dialogAndActionMemories, promptEndMemories...))
	dialogPrompt := fmt.Sprintf(
		"%s %s.\n\n%s",
		r.aiContext.AgentDescription,
		promptFormatter.FormatAnnouncedTime(time.Now()),
		memoriesAsString,
	)
	return r.complete(
		dialogPrompt,
		DefaultCompleteOptions.WithTemperature(r.textTemperature),
		memories,
		languageModel,
	)
}

// RespondToQueryWithJSON responds to the given query in the JSON format and automatically fills `obj`'s property.
func (r *ResponseService) RespondToQueryWithJSON(query string, jsonObject any) error {
	jsonQuerySchema, err := json.Marshal(jsonObject)
	if err != nil {
		return err
	}
	queryMemories := []*Memory{r.memoryFactory.NewMemory(MemoryTypeDialog, "User", query, "")}
	languageModel := r.languageModelSelector.Select(queryMemories, ResponseModeJSON)
	promptFormatter := languageModel.PromptFormatter()
	promptEndMemories := r.generatePromptEndMemories()
	memoriesAsString := promptFormatter.FormatDialog(MergeMemories(queryMemories, promptEndMemories...))
	dialogPrompt := fmt.Sprintf(
		"%s %s.\n\n%s",
		r.aiContext.AgentDescription,
		promptFormatter.FormatJSONRequest(string(jsonQuerySchema)),
		memoriesAsString,
	)
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

// generatePromptEndMemories creates a hanging "Sveta:" and the like, to make the completion engine produce the expected answer
// on the AI agent's behalf.
func (r *ResponseService) generatePromptEndMemories() []*Memory {
	agentName := r.aiContext.AgentName
	if r.aiContext.AgentDescriptionReminder != "" {
		// TODO internationalize
		agentName = fmt.Sprintf("%s (%s)", agentName, r.aiContext.AgentDescriptionReminder)
	}
	return []*Memory{
		r.memoryFactory.NewMemory(MemoryTypeDialog, agentName, "", ""),
	}
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
		if (memory.Type == MemoryTypeDialog || memory.Type == MemoryTypeAction) && memory.Who != "" {
			resultSet[memory.Who] = struct{}{}
		}
	}
	participants := make([]string, 0, len(resultSet))
	for participant := range resultSet {
		participants = append(participants, participant)
	}
	return participants
}

// For cleanResponse(..)
func getAgentNameWithDelimiter(agentName string, promptFormatter PromptFormatter) string {
	memories := []*Memory{NewMemory("", MemoryTypeAction, agentName, time.Now(), "", "", nil)}
	result := strings.TrimSpace(promptFormatter.FormatDialog(memories))
	return result
}
