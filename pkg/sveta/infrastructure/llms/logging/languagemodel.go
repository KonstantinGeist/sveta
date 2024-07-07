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

func (l *languageModelDecorator) Name() string {
	return l.wrappedLanguageModel.Name()
}

func (l *languageModelDecorator) ResponseModes() []domain.ResponseMode {
	return l.wrappedLanguageModel.ResponseModes()
}

func (l *languageModelDecorator) Complete(prompt string, options domain.CompleteOptions) (string, error) {
	l.logger.Log(fmt.Sprintf("\n================\n raw prompt (using '%s'):\n%s\n================\n\n", l.Name(), prompt))
	t := time.Now()
	response, err := l.wrappedLanguageModel.Complete(prompt, options)
	if err != nil {
		return "", err
	}
	l.logger.Log(fmt.Sprintf("\n================\n raw prompt response:\n%s\n (took %d ms)\n================\n", response, time.Now().Sub(t).Milliseconds()))
	return response, nil
}

func (l *languageModelDecorator) LegacyPromptFormatter() domain.LegacyPromptFormatter {
	return l.wrappedLanguageModel.LegacyPromptFormatter()
}

func (l *languageModelDecorator) PromptFormatter2() domain.PromptFormatter2 {
	return l.wrappedLanguageModel.PromptFormatter2()
}

func (l *languageModelDecorator) ResponseCleaner() domain.ResponseCleaner {
	return l.wrappedLanguageModel.ResponseCleaner()
}
