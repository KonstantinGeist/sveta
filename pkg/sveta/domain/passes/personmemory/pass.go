package personmemory

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"unicode/utf8"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const personMemoryCapability = "personMemories"
const maxFoundPersonMemoryCount = 5
const minKeySize = 4
const personMemoryMessage = "What %s knows about the following people:\n%s\nQuery: \"%s\" (answer using the provided information above, if they are relevant to the query, by slightly reformulating it in the language of your persona)"

var digitRemovalRegexp = regexp.MustCompile(`\d`)
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

type pass struct {
	aiContext                      *domain.AIContext
	memoryFactory                  domain.MemoryFactory
	wordFrequencyProvider          WordFrequencyProvider
	personMemories                 map[string][]string
	wordSizeThreshold              int
	wordFrequencyPositionThreshold int
}

func NewPass(
	aiContext *domain.AIContext,
	memoryFactory domain.MemoryFactory,
	wordFrequencyProvider WordFrequencyProvider,
	config *common.Config,
) domain.Pass {
	return &pass{
		aiContext:                      aiContext,
		memoryFactory:                  memoryFactory,
		wordFrequencyProvider:          wordFrequencyProvider,
		personMemories:                 loadPersonMemories(config),
		wordSizeThreshold:              config.GetIntOrDefault("personMemoryWordSizeThreshold", 2),
		wordFrequencyPositionThreshold: config.GetIntOrDefault("personMemoryWordFrequencyPositionThreshold", 4000),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        personMemoryCapability,
			Description: "takes remembered person memories into consideration when answering to the user",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(personMemoryCapability) {
		return nextPassFunc(context)
	}
	if len(p.personMemories) == 0 {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil || inputMemory.What == "" {
		return nextPassFunc(context)
	}
	what := strings.ToLower(inputMemory.What)
	if !p.shouldApply(what) {
		return nextPassFunc(context)
	}
	var foundPersonMemories []string
	for key, values := range p.personMemories {
		if strings.Contains(what, key) {
			for _, value := range values {
				foundPersonMemories = append(foundPersonMemories, fmt.Sprintf("  Information related to person \"%s\": \"%s\"", key, value))
			}
		}
	}
	if len(foundPersonMemories) == 0 {
		return nextPassFunc(context)
	}
	rand.Shuffle(len(foundPersonMemories), func(i, j int) {
		foundPersonMemories[i], foundPersonMemories[j] = foundPersonMemories[j], foundPersonMemories[i]
	})
	if len(foundPersonMemories) > maxFoundPersonMemoryCount {
		foundPersonMemories = foundPersonMemories[0:maxFoundPersonMemoryCount]
	}
	var builder strings.Builder
	for _, foundPersonMemory := range foundPersonMemories {
		builder.WriteString(foundPersonMemory)
		builder.WriteRune('\n')
	}
	what = fmt.Sprintf(personMemoryMessage, p.aiContext.AgentName, builder.String(), inputMemory.What)
	memory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, inputMemory.Who, what, inputMemory.Where)
	return nextPassFunc(context.WithMemory(domain.DataKeyInput, memory))
}

func (p *pass) shouldApply(what string) bool {
	what = strings.ReplaceAll(what, "\n", " ")
	what = strings.TrimSpace(nonAlphanumericRegex.ReplaceAllString(what, ""))
	split := strings.Split(what, " ")
	for _, word := range split {
		if utf8.RuneCountInString(word) < p.wordSizeThreshold {
			continue
		}
		position := p.wordFrequencyProvider.GetPosition(word)
		if position > p.wordFrequencyPositionThreshold || position == -1 {
			return true
		}
	}
	return false
}

func loadPersonMemories(config *common.Config) map[string][]string {
	personMemoryFilePath := config.GetStringOrDefault("personMemoryFilePath", "")
	if personMemoryFilePath == "" {
		return nil
	}
	personMemoryLines, err := common.ReadAllLines(personMemoryFilePath)
	if err != nil {
		return nil
	}
	personMemories := make(map[string][]string)
	for _, personMemoryLine := range personMemoryLines {
		split := strings.Split(personMemoryLine, "|")
		if len(split) == 2 {
			key := removeDigits(strings.ToLower(split[0]))
			if len(key) < minKeySize {
				continue
			}
			value := split[1]
			personMemories[key] = append(personMemories[key], value)
		}
	}
	return personMemories
}

func removeDigits(str string) string {
	return digitRemovalRegexp.ReplaceAllString(str, "")
}
