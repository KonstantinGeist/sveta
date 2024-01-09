package llama2

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
)

func NewLanguageModel(agentName string, config *common.Config) domain.LanguageModel {
	return llamacpp.NewLanguageModel(agentName, "llama2.bin", NewPromptFormatter(), config)
}
