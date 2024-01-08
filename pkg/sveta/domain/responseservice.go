package domain

import (
	"errors"
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
)

const unknownToken = "<unk>"
const retryCount = 3 // retries several times until the model returns an appropriate non-empty sentence

var ErrFailedToResponse = errors.New("failed to respond")

type ResponseService struct {
	agentName             string
	context               string
	responseLanguageModel LanguageModel
	embedder              Embedder
	memoryFactory         MemoryFactory
	promptFormatter       PromptFormatter
	logger                common.Logger
}

func NewResponseService(
	agentName string,
	responseLanguageModel LanguageModel,
	embedder Embedder,
	memoryFactory MemoryFactory,
	promptFormatter PromptFormatter,
	logger common.Logger,
) *ResponseService {
	return &ResponseService{
		agentName:             agentName,
		responseLanguageModel: responseLanguageModel,
		embedder:              embedder,
		memoryFactory:         memoryFactory,
		promptFormatter:       promptFormatter,
		logger:                logger,
	}
}

func (r *ResponseService) RespondToMemories(memories []*Memory) (string, error) {
	if len(memories) == 0 {
		return "", nil
	}
	dialogAndActionMemories := FilterMemoriesByTypes(memories, []MemoryType{MemoryTypeDialog, MemoryTypeAction})
	promptEndMemories := r.generatePromptEndMemories()
	memoriesAsString := r.promptFormatter.FormatDialog(MergeMemories(dialogAndActionMemories, promptEndMemories...))
	dialogPrompt := fmt.Sprintf(
		"%s\n\n%s",
		r.context,
		memoriesAsString,
	)
	return r.complete(dialogPrompt, memories, r.responseLanguageModel)
}

func (r *ResponseService) SetContext(context string) error {
	r.context = context
	return nil
}

func (r *ResponseService) generatePromptEndMemories() []*Memory {
	return []*Memory{
		r.memoryFactory.NewMemory(MemoryTypeDialog, r.agentName, "", ""),
	}
}

func (r *ResponseService) complete(prompt string, memories []*Memory, languageModel LanguageModel) (string, error) {
	if len(memories) == 0 {
		return "", ErrFailedToResponse
	}
	for i := 0; i < retryCount; i++ {
		response, err := languageModel.Complete(cleanPrompt(prompt))
		if err != nil {
			return "", err
		}
		// Sometimes, the model can return an "unknown" token, skip such responses, as they're broken.
		if strings.Contains(response, unknownToken) {
			continue
		}
		dialogParticipants := r.collectDialogParticipants(memories)
		cleanResponse := r.cleanResponse(response, dialogParticipants)
		// Sometimes, a model can just repeat the user's name.
		if strings.ToLower(cleanResponse) == strings.ToLower(memories[len(memories)-1].Who) {
			continue
		}
		if cleanResponse != "" {
			return cleanResponse, nil
		}
	}
	return "", ErrFailedToResponse
}

// Sometimes, the model can generate too much (for example, trying to complete other participants' dialogs), so we trim it.
// TODO move to response cleaner
func (r *ResponseService) cleanResponse(response string, participants []string) string {
	agentNamePrefix := r.agentName + ":"
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, agentNamePrefix) {
		response = response[len(agentNamePrefix):]
	}
	for _, participant := range participants {
		participantPrefix := participant + ":"
		foundIndex := strings.Index(response, participantPrefix)
		if foundIndex > 0 { // in the middle/at the end, when it wants to keep generating the dialog
			response = response[0:foundIndex]
		} else if foundIndex == 0 { // in the beginning, like: "User: "
			response = response[len(participantPrefix):]
		}
	}
	return unquoteResponse(strings.TrimSpace(response))
}

func (r *ResponseService) collectDialogParticipants(memories []*Memory) []string {
	resultSet := make(map[string]struct{})
	resultSet[r.agentName] = struct{}{}
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

func (r *ResponseService) getLastWho(memories []*Memory) string {
	lastWho := "Narrator"
	for _, memory := range memories {
		if memory.Who != r.agentName {
			lastWho = memory.Who
		}
	}
	return lastWho
}

func cleanPrompt(prompt string) string {
	// Sometimes, quotes mess up completions, so we remove them so far.
	//prompt = strings.ReplaceAll(prompt, "\"", "")
	return prompt
}

func unquoteResponse(response string) string {
	quoteCount := strings.Count(response, "\"")
	// For mismatched quotes like "Hello
	if quoteCount == 1 {
		response = strings.ReplaceAll(response, "\"", "")
	}
	// For direct speech like "Hello"
	if quoteCount == 2 && len(response) > 2 && response[0] == '"' && response[len(response)-1] == '"' {
		response = response[1 : len(response)-1]
	}
	return strings.TrimSpace(response)
}
