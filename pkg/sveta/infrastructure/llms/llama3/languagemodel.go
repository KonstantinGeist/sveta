package llama3

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
)

func NewLanguageModel(config *common.Config, logger common.Logger) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		"llama3",
		"llama3.bin",
		[]domain.ResponseMode{domain.ResponseModeNormal, domain.ResponseModeJSON},
		newPromptFormatter(),
		newStopCondition(),
		newResponseCleaner(),
		config,
		logger,
	)
}
