package solar

import (
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/common"
)

type promptFormatter struct{}

func NewPromptFormatter() domain.PromptFormatter {
	return &promptFormatter{}
}

func (p *promptFormatter) FormatDialog(memories []*domain.Memory) string {
	return common.FormatAsAlpacaDialog(memories)
}

func (p *promptFormatter) FormatAnnouncedTime(t time.Time) string {
	return common.FormatAnnouncedTimeInEnglish(t)
}

func (p *promptFormatter) FormatJSONRequest(jsonSchemaQuery string) string {
	return common.FormatJSONRequestInEnglish(jsonSchemaQuery)
}
