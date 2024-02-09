package filesystem

import (
	"strings"

	"kgeyst.com/sveta/pkg/common"
)

// A very crude way to estimate positions of derivative words.
// NOTE: only supports English
var commonWordEndings = []string{
	"d",
	"ed",
	"s",
	"es",
	"ing",
	"in",
	"ings",
}

type WordFrequencyProvider struct {
	words map[string]int
}

func NewWordFrequencyProvider(
	config *common.Config,
	logger common.Logger,
) *WordFrequencyProvider {
	filePath := config.GetStringOrDefault("wordFrequencyFilePath", "word_frequencies.txt")
	lines, err := common.ReadAllLines(filePath)
	words := make(map[string]int)
	if err == nil {
		for index, line := range lines {
			words[line] = index
		}
	} else {
		logger.Log("failed to load word frequency list: " + err.Error())
	}
	return &WordFrequencyProvider{
		words: words,
	}
}

func (w *WordFrequencyProvider) GetPosition(word string) int {
	word = strings.ToLower(word)
	frequency, ok := w.words[word]
	if ok {
		return frequency
	}
	for _, commonWordEnding := range commonWordEndings {
		if !strings.HasSuffix(word, commonWordEnding) {
			continue
		}
		wordWithoutEnding := word[0 : len(word)-len(commonWordEnding)]
		frequency, ok = w.words[wordWithoutEnding]
		if ok {
			return frequency
		}
	}
	return -1
}
