package llama2

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
)

func NewRoleplayLanguageModel(aiContext *domain.AIContext, config *common.Config, logger common.Logger) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		aiContext,
		"llama2-roleplay",
		"llama2-roleplay.bin",
		[]domain.ResponseMode{domain.ResponseModeNormal},
		NewPromptFormatter(),
		config,
		logger,
	)
}
