package api

import (
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/embed4all"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/inmemory"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llama"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/logging"
)

type api struct {
	agent *domain.AIService
}

// See domain/config.go
const (
	ConfigKeyAgentName = domain.ConfigKeyAgentName
	ConfigKeyContext   = domain.ConfigKeyContext
	ConfigKeyLogPath   = domain.ConfigKeyLogPath
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
	// LoadMemory loads a precomputed vector store of a document (specified by `path`) so that the AI could use RAG
	// ("retrieval-augmented generation") to answer to questions not found in the original model.
	// The vector store's data is ingested using bin/embed_corpus.py
	LoadMemory(path, who, where string, when time.Time) error
	// ForgetEverything makes the AI forget all current context across all rooms. Useful for debugging.
	// Note that it removes all memory loaded previously with LoadMemory.
	ForgetEverything() error
	// SetContext resets the context ("system prompt") of the AI. Useful for debugging.
	SetContext(context string) error
}

func NewAPI(config *common.Config) API {
	logger := common.NewFileLogger(config.GetStringOrDefault(ConfigKeyLogPath, "log.txt"))
	embedder := embed4all.NewEmbedder()
	agentName := config.GetStringOrDefault(ConfigKeyAgentName, "Sveta")
	responseModel := logging.NewLanguageModelDecorator(llama.NewLanguageModel(agentName, config), logger)
	promptFormatter := llama.NewPromptFormatter()
	memoryRepository := inmemory.NewMemoryRepository()
	memoryFactory := inmemory.NewMemoryFactory(memoryRepository, embedder)
	return &api{
		agent: domain.NewAIService(
			agentName,
			memoryRepository,
			memoryFactory,
			domain.NewResponseService(
				agentName,
				responseModel,
				embedder,
				memoryFactory,
				promptFormatter,
				config,
				logger,
			),
			promptFormatter,
			logger,
			config,
		),
	}
}

func (a *api) Respond(who string, what string, where string) (string, error) {
	return a.agent.Respond(who, what, where)
}

func (a *api) RememberAction(who string, what string, where string) error {
	return a.agent.RememberAction(who, what, where)
}

func (a *api) LoadMemory(path, who, where string, when time.Time) error {
	return a.agent.LoadMemory(path, who, where, when)
}

func (a *api) ForgetEverything() error {
	return a.agent.ForgetEverything()
}

func (a *api) SetContext(context string) error {
	return a.agent.SetContext(context)
}
