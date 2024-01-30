package solar

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/llamacpp"
)

func NewGenericLanguageModel(aiContext *domain.AIContext, config *common.Config) domain.LanguageModel {
	return llamacpp.NewLanguageModel(aiContext, "solar-generic", "solar-generic.bin", domain.LanguageModelPurposeGeneric, NewPromptFormatter(), config)
}
