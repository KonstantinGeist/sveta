package llama2

import (
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type promptFormatter struct{}

func NewPromptFormatter() domain.PromptFormatter {
	return &promptFormatter{}
}

func (p *promptFormatter) FormatDialog(memories []*domain.Memory) string {
	var buf strings.Builder
	for i := 0; i < len(memories); i++ {
		memory := memories[i]
		buf.WriteString("### " + memory.Who)
		buf.WriteString(":\n")
		buf.WriteString(memory.What)
		if i < len(memories)-1 {
			buf.WriteString("\n\n")
		}
	}
	return buf.String()
}
