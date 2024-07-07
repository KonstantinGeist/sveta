package solar

import (
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/common"
)

type LegacyPromptFormatter struct{}

func NewLegacyPromptFormatter() *LegacyPromptFormatter {
	return &LegacyPromptFormatter{}
}

func (p *LegacyPromptFormatter) FormatDialog(memories []*domain.Memory) string {
	return common.FormatAsAlpacaDialog(memories)
}

func (p *LegacyPromptFormatter) FormatAnnouncedTime(t time.Time) string {
	return common.FormatAnnouncedTimeInEnglish(t)
}

func (p *LegacyPromptFormatter) FormatJSONRequest(jsonSchemaQuery string) string {
	return common.FormatJSONRequestInEnglish(jsonSchemaQuery)
}
