package deepseekcoder

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llamacpp"
)

func NewLanguageModel(
	namedMutexAcquirer domain.NamedMutexAcquirer,
	config *common.Config,
	logger common.Logger,
) *llamacpp.LanguageModel {
	return llamacpp.NewLanguageModel(
		"deepseekcoder",
		"deepseekcoder.bin",
		[]domain.ResponseMode{domain.ResponseModeCode},
		newPromptFormatter(),
		newStopCondition(),
		newResponseCleaner(),
		namedMutexAcquirer,
		config,
		logger,
	)
}
