package llama2

import (
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/common"
)

type PromptFormatter struct{}

func NewPromptFormatter() *PromptFormatter {
	return &PromptFormatter{}
}

func (p *PromptFormatter) FormatDialog(memories []*domain.Memory) string {
	return common.FormatAsAlpacaDialog(memories)
}

func (p *PromptFormatter) FormatAnnouncedTime(t time.Time) string {
	// TODO internationalize here and in other such instances as well
	return common.FormatAnnouncedTimeInEnglish(t)
}

func (p *PromptFormatter) FormatJSONRequest(jsonSchemaQuery string) string {
	return common.FormatJSONRequestInEnglish(jsonSchemaQuery)
}
