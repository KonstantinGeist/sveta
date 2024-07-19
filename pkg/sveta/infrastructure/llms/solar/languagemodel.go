package solar

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
	llmscommon "kgeyst.com/sveta/pkg/sveta/infrastructure/llms/common"
)

func NewGenericLanguageModel(
	aiContext *domain.AIContext,
	namedMutexAcquirer domain.NamedMutexAcquirer,
	config *common.Config,
	logger common.Logger,
) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		"solar-generic",
		"solar-generic.bin",
		[]domain.ResponseMode{domain.ResponseModeNormal, domain.ResponseModeRerank, domain.ResponseModeJSON},
		llmscommon.NewAlpacaPromptFormatter(),
		llmscommon.NewAlpacaStopCondition(aiContext),
		llmscommon.NewAlpacaResponseCleaner(),
		namedMutexAcquirer,
		config,
		logger,
	)
}
