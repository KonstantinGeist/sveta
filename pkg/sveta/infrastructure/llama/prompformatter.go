package llama

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type promptFormatter struct {
	agentName string
}

func NewPromptFormatter(agentName string) domain.PromptFormatter {
	return &promptFormatter{
		agentName: agentName,
	}
}

func (p *promptFormatter) FormatDialog(memories []*domain.Memory) string {
	var buf strings.Builder
	for i := 0; i < len(memories); i++ {
		memory := memories[i]
		buf.WriteString(memory.Who)
		buf.WriteString(":\n")
		buf.WriteString(memory.What)
		if i < len(memories)-1 {
			buf.WriteString("\n\n")
		}
	}
	return buf.String()
}

func (p *promptFormatter) FormatSummary(context string, summaryMemories []*domain.Memory) string {
	var summaries []string
	for _, memory := range summaryMemories {
		summaries = append(summaries, memory.What)
	}
	summary := strings.TrimSpace(strings.Join(summaries, ". "))
	if summary != "" {
		summary = fmt.Sprintf("%s Quick recap of the conversation: %s.", context, summary)
	} else {
		summary = context
	}
	return summary
}

func (p *promptFormatter) GetSummaryPrompt() string {
	return "I want you to make a very short, concise summary of our conversation."
}
