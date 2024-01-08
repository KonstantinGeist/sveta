package domain

type PromptFormatter interface {
	FormatDialog(memories []*Memory) string
	FormatSummary(context string, summaryMemories []*Memory) string
	GetSummaryPrompt() string
}
