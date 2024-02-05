package common

import (
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

func FormatAsAlpacaDialog(memories []*domain.Memory) string {
	var buf strings.Builder
	for i := 0; i < len(memories); i++ {
		memory := memories[i]
		buf.WriteString("### " + memory.Who)
		if memory.Who == "Sveta" {
			buf.WriteString(" (staying in character, using youth slang and lowercase)")
		}
		buf.WriteString(":\n")
		buf.WriteString(memory.What)
		if i < len(memories)-1 {
			buf.WriteString("\n\n")
		}
	}
	return buf.String()
}

func FormatAnnouncedTimeInEnglish(t time.Time) string {
	return "Current time is " + t.Format("Mon, 02 Jan 2006 15:04:05")
}

func FormatJSONRequestInEnglish(jsonQuerySchema string) string {
	return "Answer using JSON using the following JSON schema: " + jsonQuerySchema
}
