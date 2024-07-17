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
	wordsToPositions map[string]int
	positionsToWords map[int]string
}

func NewWordFrequencyProvider(
	config *common.Config,
	logger common.Logger,
) *WordFrequencyProvider {
	filePath := config.GetStringOrDefault("wordFrequencyFilePath", "word_frequencies.txt")
	lines, err := common.ReadAllLines(filePath)
	wordsToPositions := make(map[string]int)
	positionsToWords := make(map[int]string)
	if err == nil {
		for index, line := range lines {
			wordsToPositions[line] = index
			positionsToWords[index] = line
		}
	} else {
		logger.Log("failed to load word frequency list: " + err.Error())
	}
	return &WordFrequencyProvider{
		wordsToPositions: wordsToPositions,
		positionsToWords: positionsToWords,
	}
}

func (w *WordFrequencyProvider) GetPosition(word string) int {
	word = strings.ToLower(word)
	frequency, ok := w.wordsToPositions[word]
	if ok {
		return frequency
	}
	for _, commonWordEnding := range commonWordEndings {
		if !strings.HasSuffix(word, commonWordEnding) {
			continue
		}
		wordWithoutEnding := word[0 : len(word)-len(commonWordEnding)]
		frequency, ok = w.wordsToPositions[wordWithoutEnding]
		if ok {
			return frequency
		}
	}
	return -1
}

func (w *WordFrequencyProvider) MaxPosition() int {
	return len(w.wordsToPositions)
}

func (w *WordFrequencyProvider) GetWordAtPosition(position int) string {
	return w.positionsToWords[position]
}
