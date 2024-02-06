package solar

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
)

func NewGenericLanguageModel(aiContext *domain.AIContext, config *common.Config, logger common.Logger) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		aiContext,
		"solar-generic",
		"solar-generic.bin",
		[]domain.ResponseMode{domain.ResponseModeNormal, domain.ResponseModeJSON, domain.ResponseModeRerank},
		NewPromptFormatter(),
		config,
		logger,
	)
}
