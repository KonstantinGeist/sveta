package notes

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const notesCapability = "notes"
const maxFoundNoteCount = 5
const minKeySize = 4
const notesMessage = "Notes found relevant to our discussion:\n%s\nQuery: \"%s\" (answer using the provided notes above, but ONLY if they are relevant to the query, by slightly reformulating it in the language of your persona)"

var digitRemovalRegexp = regexp.MustCompile(`\d`)

type pass struct {
	memoryFactory domain.MemoryFactory
	notes         map[string][]string
}

func NewPass(
	memoryFactory domain.MemoryFactory,
	config *common.Config,
) domain.Pass {
	return &pass{
		memoryFactory: memoryFactory,
		notes:         loadNotes(config),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        notesCapability,
			Description: "takes remembered notes into consideration when answering to the user",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(notesCapability) {
		return nextPassFunc(context)
	}
	if len(p.notes) == 0 {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil || inputMemory.What == "" {
		return nextPassFunc(context)
	}
	what := strings.ToLower(inputMemory.What)
	var foundNotes []string
	for key, values := range p.notes {
		if strings.Contains(what, key) {
			for _, value := range values {
				foundNotes = append(foundNotes, fmt.Sprintf("  A note about \"%s\": \"%s\"", key, value))
			}
		}
	}
	if len(foundNotes) == 0 {
		return nextPassFunc(context)
	}
	rand.Shuffle(len(foundNotes), func(i, j int) {
		foundNotes[i], foundNotes[j] = foundNotes[j], foundNotes[i]
	})
	if len(foundNotes) > maxFoundNoteCount {
		foundNotes = foundNotes[0:maxFoundNoteCount]
	}
	var builder strings.Builder
	for _, foundNote := range foundNotes {
		builder.WriteString(foundNote)
		builder.WriteRune('\n')
	}
	what = fmt.Sprintf(notesMessage, builder.String(), inputMemory.What)
	memory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, inputMemory.Who, what, inputMemory.Where)
	return nextPassFunc(context.WithMemory(domain.DataKeyInput, memory))
}

func loadNotes(config *common.Config) map[string][]string {
	notesFilePath := config.GetStringOrDefault("notesFilePath", "")
	if notesFilePath == "" {
		return nil
	}
	noteLines, err := common.ReadAllLines("notes.txt")
	if err != nil {
		return nil
	}
	notes := make(map[string][]string)
	for _, noteLine := range noteLines {
		split := strings.Split(noteLine, "|")
		if len(split) == 2 {
			key := removeDigits(strings.ToLower(split[0]))
			if len(key) < minKeySize {
				continue
			}
			value := split[1]
			notes[key] = append(notes[key], value)
		}
	}
	return notes
}

func removeDigits(str string) string {
	return digitRemovalRegexp.ReplaceAllString(str, "")
}
