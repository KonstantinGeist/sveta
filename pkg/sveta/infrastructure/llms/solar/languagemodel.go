package solar

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
	llmscommon "kgeyst.com/sveta/pkg/sveta/infrastructure/llms/common"
)

func NewGenericLanguageModel(aiContext *domain.AIContext, config *common.Config, logger common.Logger) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		"solar-generic",
		"solar-generic.bin",
		[]domain.ResponseMode{domain.ResponseModeNormal, domain.ResponseModeJSON, domain.ResponseModeRerank},
		llmscommon.NewAlpacataPromptFormatter(),
		llmscommon.NewAlpacaStopCondition(aiContext),
		llmscommon.NewAlpacaResponseCleaner(),
		config,
		logger,
	)
}
