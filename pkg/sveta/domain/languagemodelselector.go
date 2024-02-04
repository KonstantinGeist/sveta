package domain

// LanguageModelSelector makes sure the right language model is chosen for a given scenario.
type LanguageModelSelector struct {
	responseModesToLanguageModels       map[ResponseMode][]LanguageModel
	responseModesToLanguageModelIndices map[ResponseMode]int
}

func NewLanguageModelSelector(languageModels []LanguageModel) *LanguageModelSelector {
	modesToLanguageModels := make(map[ResponseMode][]LanguageModel)
	modesToLanguageModelIndices := make(map[ResponseMode]int)
	for _, languageModel := range languageModels {
		for _, modes := range languageModel.Modes() {
			modesToLanguageModels[modes] = append(modesToLanguageModels[modes], languageModel)
		}
	}
	for _, responseMode := range ResponseModes {
		modesToLanguageModelIndices[responseMode] = 0
	}
	return &LanguageModelSelector{
		responseModesToLanguageModels:       modesToLanguageModels,
		responseModesToLanguageModelIndices: modesToLanguageModelIndices,
	}
}

// Select given a list of memories and the response mode, finds the language model most suitable for the task.
// TODO not thread-safe
func (l *LanguageModelSelector) Select(_ []*Memory, responseMode ResponseMode) LanguageModel {
	languageModelIndex := l.responseModesToLanguageModelIndices[responseMode]
	languageModel := l.responseModesToLanguageModels[responseMode][languageModelIndex]
	l.responseModesToLanguageModelIndices[responseMode] = (languageModelIndex + 1) % (len(l.responseModesToLanguageModels[responseMode]))
	return languageModel
}
