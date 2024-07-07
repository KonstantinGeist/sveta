package common

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type AlpacaResponseCleaner struct{}

func NewAlpacaResponseCleaner() *AlpacaResponseCleaner {
	return &AlpacaResponseCleaner{}
}

func (a *AlpacaResponseCleaner) CleanResponse(options domain.CleanOptions) string {
	agentNamePrefix := a.getNameWithDelimiter(options.AgentName)
	response := strings.TrimSpace(options.Response)
	if strings.HasPrefix(response, agentNamePrefix) {
		response = response[len(agentNamePrefix):]
	}
	participants := a.collectDialogParticipants(options)
	for _, participant := range participants {
		participantPrefix := a.getNameWithDelimiter(participant)
		foundIndex := strings.Index(response, participantPrefix)
		if foundIndex > 0 { // in the middle/at the end, when it wants to keep generating the dialog
			response = response[0:foundIndex]
		} else if foundIndex == 0 { // in the beginning, like: "User: "
			response = response[len(participantPrefix):]
		}
	}
	return strings.TrimSpace(response)
}

func (a *AlpacaResponseCleaner) collectDialogParticipants(options domain.CleanOptions) []string {
	resultSet := make(map[string]struct{})
	resultSet[options.AgentName] = struct{}{}
	for _, memory := range options.Memories {
		if (memory.Type == domain.MemoryTypeDialog) && memory.Who != "" {
			resultSet[memory.Who] = struct{}{}
		}
	}
	participants := make([]string, 0, len(resultSet))
	for participant := range resultSet {
		participants = append(participants, participant)
	}
	return participants
}

func (a *AlpacaResponseCleaner) getNameWithDelimiter(name string) string {
	return fmt.Sprintf("### %s:", name)
}
