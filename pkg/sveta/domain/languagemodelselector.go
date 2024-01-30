package domain

// LanguageModelSelector makes sure the right language model is chosen for a given scenario.
type LanguageModelSelector struct {
	allLanguageModels       []LanguageModel
	jsonLanguageModels      []LanguageModel
	allLanguageModelsIndex  int
	jsonLanguageModelsIndex int
}

func NewLanguageModelSelector(languageModels []LanguageModel) *LanguageModelSelector {
	var jsonLanguageModels []LanguageModel
	for _, languageModel := range languageModels {
		if languageModel.Purpose() == LanguageModelPurposeGeneric ||
			languageModel.Purpose() == LanguageModelPurposeJSON {
			jsonLanguageModels = append(jsonLanguageModels, languageModel)
		}
	}
	return &LanguageModelSelector{
		allLanguageModels:  languageModels,
		jsonLanguageModels: jsonLanguageModels,
	}
}

// Select given a list of memories and the parameter `jsonMode`, finds the language model most suitable for the task.
// TODO not thread-safe
func (l *LanguageModelSelector) Select(_ []*Memory, jsonMode bool) LanguageModel {
	if jsonMode {
		languageModel := l.jsonLanguageModels[l.jsonLanguageModelsIndex]
		l.jsonLanguageModelsIndex = (l.jsonLanguageModelsIndex + 1) % len(l.jsonLanguageModels)
		return languageModel
	}
	languageModel := l.allLanguageModels[l.allLanguageModelsIndex]
	l.allLanguageModelsIndex = (l.allLanguageModelsIndex + 1) % len(l.allLanguageModels)
	return languageModel
}
