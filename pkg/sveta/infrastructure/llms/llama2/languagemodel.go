package llama2

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
	llmscommon "kgeyst.com/sveta/pkg/sveta/infrastructure/llms/common"
)

func NewRoleplayLanguageModel(
	aiContext *domain.AIContext,
	namedMutexAcquirer domain.NamedMutexAcquirer,
	config *common.Config,
	logger common.Logger,
) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		"llama2-roleplay",
		"llama2-roleplay.bin",
		[]domain.ResponseMode{domain.ResponseModeNormal},
		llmscommon.NewAlpacaPromptFormatter(),
		llmscommon.NewAlpacaStopCondition(aiContext),
		llmscommon.NewAlpacaResponseCleaner(),
		namedMutexAcquirer,
		config,
		logger,
	)
}
