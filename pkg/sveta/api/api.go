package api

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	domainwiki "kgeyst.com/sveta/pkg/sveta/domain/aifilters/wiki"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/aifilters"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/embed4all"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/inmemory"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/llama2"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/logging"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/solar"
	infrawiki "kgeyst.com/sveta/pkg/sveta/infrastructure/wiki"
)

type api struct {
	aiService *domain.AIService
}

// See domain/config.go
const (
	ConfigKeyAgentName        = domain.ConfigKeyAgentName
	ConfigKeyAgentDescription = domain.ConfigKeyAgentDescription
	ConfigKeyLogPath          = domain.ConfigKeyLogPath
)

// API is the entrypoint to Sveta. It shouldn't contain any logic of its own; it glues all the components together
// and provides a public interface for domain.AIService.
// This API can be used in various contexts: in an IRC chat, an HTTP server, console input/output etc.
type API interface {
	// Respond makes Sveta respond to the given prompt (`what`). Parameter `who` specifies the user (so that Sveta could
	// tell between users in a shared chat and could respond intelligently). Parameter `where` specifies a shared virtual "room"
	// (useful for isolating dialogs from each other).
	Respond(who string, what string, where string) (string, error)
	// RememberAction remembers a certain action (not a reply) in the chat: for example, that a certain user entered the chat.
	// The AI can use this information for enriching the context of the dialog.
	RememberAction(who string, what string, where string) error
	// RememberDialog remembers a certain utterance in the chat. The AI can use this information for enriching the context
	// of the dialog without directly responding to it (as is usual with Respond(..)
	RememberDialog(who string, what string, where string) error
	// ClearAllMemory makes the AI forget all current context across all rooms. Useful for debugging.
	// Note that it removes all memory loaded previously with LoadMemory.
	ClearAllMemory() error
	// ChangeAgentDescription resets the context ("system prompt") of the AI. Useful for debugging.
	ChangeAgentDescription(context string) error
}

func NewAPI(config *common.Config) API {
	logger := common.NewFileLogger(config.GetStringOrDefault(ConfigKeyLogPath, "log.txt"))
	embedder := embed4all.NewEmbedder()
	aiContext := domain.NewAIContextFromConfig(config)
	roleplayLLama2Model := logging.NewLanguageModelDecorator(llama2.NewRoleplayLanguageModel(aiContext, config), logger)
	genericSolarModel := logging.NewLanguageModelDecorator(solar.NewGenericLanguageModel(aiContext, config), logger)
	languageModelSelector := domain.NewLanguageModelSelector([]domain.LanguageModel{genericSolarModel, roleplayLLama2Model})
	memoryRepository := inmemory.NewMemoryRepository()
	memoryFactory := inmemory.NewMemoryFactory(memoryRepository, embedder)
	responseService := domain.NewResponseService(
		aiContext,
		languageModelSelector,
		embedder,
		memoryFactory,
		config,
		logger,
	)
	promptFormatterForLog := llama2.NewPromptFormatter()
	newsFilter := aifilters.NewNewsFilter(memoryRepository, memoryFactory, config, logger)
	htmlFilter := aifilters.NewHTMLFilter(config, logger)
	imageFilter := aifilters.NewImageFilter(config, logger)
	wikiFilter := domainwiki.NewWikiFilter(
		responseService,
		memoryFactory,
		memoryRepository,
		infrawiki.NewArticleProvider(),
		logger,
		config,
	)
	responseFilter := domain.NewResponseFilter(
		aiContext,
		memoryFactory,
		memoryRepository,
		responseService,
		embedder,
		promptFormatterForLog,
		logger,
		config,
	)
	return &api{
		aiService: domain.NewAIService(
			memoryRepository,
			memoryFactory,
			aiContext,
			[]domain.AIFilter{
				newsFilter,
				htmlFilter,
				imageFilter,
				wikiFilter,
				responseFilter,
			},
		),
	}
}

func (a *api) Respond(who string, what string, where string) (string, error) {
	return a.aiService.Respond(who, what, where)
}

func (a *api) RememberAction(who string, what string, where string) error {
	return a.aiService.RememberAction(who, what, where)
}

func (a *api) RememberDialog(who string, what string, where string) error {
	return a.aiService.RememberDialog(who, what, where)
}

func (a *api) ClearAllMemory() error {
	return a.aiService.ClearAllMemory()
}

func (a *api) ChangeAgentDescription(context string) error {
	return a.aiService.ChangeAgentDescription(context)
}
