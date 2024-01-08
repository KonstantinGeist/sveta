package logging

import (
	"fmt"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type languageModelDecorator struct {
	wrappedLanguageModel domain.LanguageModel
	logger               common.Logger
}

func NewLanguageModelDecorator(wrappedLanguageModel domain.LanguageModel, logger common.Logger) domain.LanguageModel {
	return &languageModelDecorator{
		wrappedLanguageModel: wrappedLanguageModel,
		logger:               logger,
	}
}

func (l *languageModelDecorator) Complete(prompt string) (string, error) {
	l.logger.Log(fmt.Sprintf("\n================\n raw prompt:\n%s\n================\n\n", prompt))
	t := time.Now()
	response, err := l.wrappedLanguageModel.Complete(prompt)
	if err != nil {
		return "", err
	}
	l.logger.Log(fmt.Sprintf("\n================\n raw prompt response:\n%s\n (took %d ms)\n================\n", response, time.Now().Sub(t).Milliseconds()))
	return response, nil
}
