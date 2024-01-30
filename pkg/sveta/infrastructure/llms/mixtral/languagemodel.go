package mixtral

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
)

func NewGenericLanguageModel(aiContext *domain.AIContext, config *common.Config) domain.LanguageModel {
	return llamacpp.NewLanguageModel(aiContext, "mixtral-generic", "mixtral-generic.bin", domain.LanguageModelPurposeGeneric, NewPromptFormatter(), config)
}
