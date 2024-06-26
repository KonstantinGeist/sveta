package api

import (
	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/bio"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/facts"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/function"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/news"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/personmemory"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/remember"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/response"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/rewrite"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/summary"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/vision"
	domainweb "kgeyst.com/sveta/pkg/sveta/domain/passes/web"
	domainwiki "kgeyst.com/sveta/pkg/sveta/domain/passes/wiki"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/workingmemory"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/embed4all"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/filesystem"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/inmemory"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llavacpp"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/llama2"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/logging"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llms/solar"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/rss"
	infraweb "kgeyst.com/sveta/pkg/sveta/infrastructure/web"
	infrawiki "kgeyst.com/sveta/pkg/sveta/infrastructure/wiki"
)

type FunctionDesc = domain.FunctionDesc
type FunctionInput = domain.FunctionInput
type FunctionOutput = domain.FunctionOutput

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
	// RememberDialog remembers a certain utterance in the chat. The AI can use this information for enriching the context
	// of the dialog without directly responding to it (as is usual with Respond(..)
	RememberDialog(who string, what string, where string) error
	// ClearAllMemory makes the AI forget all current context across all rooms. Useful for debugging.
	// Note that it removes all memory loaded previously with LoadMemory.
	ClearAllMemory() error
	// ChangeAgentDescription resets the context ("system prompt") of the AI. Useful for debugging.
	ChangeAgentDescription(description string) error
	ChangeAgentName(name string) error
	GetSummary(where string) (string, error)
	RegisterFunction(functionDesc FunctionDesc) error
	ListCapabilities() []string
	EnableCapability(name string, value bool) error
}

func NewAPI(config *common.Config) (API, common.Stopper) {
	logger := common.NewFileLogger(config.GetStringOrDefault(ConfigKeyLogPath, "sveta.log"))
	languageModelJobQueue := common.NewJobQueue(logger)
	tempFileProvider := filesystem.NewTempFilePathProvider(config)
	embedder := embed4all.NewEmbedder(logger)
	aiContext := domain.NewAIContextFromConfig(config)
	roleplayLLama2Model := logging.NewLanguageModelDecorator(llama2.NewRoleplayLanguageModel(aiContext, config, logger), logger)
	genericSolarModel := logging.NewLanguageModelDecorator(solar.NewGenericLanguageModel(aiContext, config, logger), logger)
	languageModelSelector := domain.NewLanguageModelSelector([]domain.LanguageModel{genericSolarModel, roleplayLLama2Model})
	inMemoryMemoryRepository := inmemory.NewMemoryRepository()
	memoryRepository := filesystem.NewMemoryRepository(inMemoryMemoryRepository, config, logger)
	memoryFactory := inmemory.NewMemoryFactory(memoryRepository, embedder)
	summaryRepository := inmemory.NewSummaryRepository()
	responseService := domain.NewResponseService(
		aiContext,
		languageModelSelector,
		embedder,
		memoryFactory,
		summaryRepository,
		config,
		logger,
	)
	functionService := domain.NewFunctionService(aiContext, responseService)
	promptFormatterForLog := llama2.NewPromptFormatter()
	urlFinder := infraweb.NewURLFinder()
	newsProvider := rss.NewNewsProvider(
		config.GetStringOrDefault("newsSourceURL", "http://www.independent.co.uk/rss"),
	)
	workingMemoryPass := workingmemory.NewPass(
		memoryRepository,
		memoryFactory,
		config,
		logger,
	)
	rewritePass := rewrite.NewPass(
		memoryFactory,
		responseService,
		logger,
	)
	newsPass := news.NewPass(
		newsProvider,
		memoryRepository,
		memoryFactory,
		summaryRepository,
		config,
		logger,
	)
	bioPass := bio.NewPass(
		aiContext,
		filesystem.NewBioFactProvider(config),
		memoryRepository,
		memoryFactory,
		logger,
	)
	webPass := domainweb.NewPass(
		urlFinder,
		infraweb.NewPageContentExtractor(),
		config,
		logger,
	)
	visionModel := llavacpp.NewVisionModel()
	visionPass := vision.NewPass(
		urlFinder,
		visionModel,
		tempFileProvider,
		config,
		logger,
	)
	wordFrequencyProvider := filesystem.NewWordFrequencyProvider(config, logger)
	wikiPass := domainwiki.NewPass(
		responseService,
		memoryFactory,
		memoryRepository,
		infrawiki.NewArticleProvider(),
		wordFrequencyProvider,
		config,
		logger,
	)
	functionPass := function.NewPass(memoryFactory, functionService, logger)
	personMemoryPass := personmemory.NewPass(
		aiContext,
		memoryFactory,
		wordFrequencyProvider,
		config,
	)
	responsePass := response.NewPass(
		aiContext,
		memoryFactory,
		memoryRepository,
		responseService,
		embedder,
		promptFormatterForLog,
		config,
		logger,
	)
	rememberPass := remember.NewPass(memoryRepository)
	summaryPass := summary.NewPass(
		aiContext,
		summaryRepository,
		responseService,
		languageModelJobQueue,
		logger,
	)
	factsPass := facts.NewPass(
		aiContext,
		memoryRepository,
		memoryFactory,
		responseService,
		languageModelJobQueue,
		logger,
	)
	return &api{
		aiService: domain.NewAIService(
			memoryRepository,
			memoryFactory,
			summaryRepository,
			functionService,
			aiContext,
			[]domain.Pass{
				workingMemoryPass,
				newsPass,
				bioPass,
				rewritePass,
				webPass,
				visionPass,
				wikiPass,
				functionPass,
				personMemoryPass,
				responsePass,
				rememberPass,
				summaryPass,
				factsPass,
			},
		),
	}, languageModelJobQueue
}

func (a *api) Respond(who string, what string, where string) (string, error) {
	return a.aiService.Respond(who, what, where)
}

func (a *api) RememberDialog(who string, what string, where string) error {
	return a.aiService.RememberDialog(who, what, where)
}

func (a *api) ClearAllMemory() error {
	return a.aiService.ClearAllMemory()
}

func (a *api) ChangeAgentDescription(description string) error {
	return a.aiService.ChangeAgentDescription(description)
}

func (a *api) ChangeAgentName(name string) error {
	return a.aiService.ChangeAgentName(name)
}

func (a *api) GetSummary(where string) (string, error) {
	return a.aiService.GetSummary(where)
}

func (a *api) RegisterFunction(functionDesc FunctionDesc) error {
	return a.aiService.RegisterFunction(functionDesc)
}

func (a *api) ListCapabilities() []string {
	return a.aiService.ListCapabilities()
}

func (a *api) EnableCapability(name string, value bool) error {
	return a.aiService.EnableCapability(name, value)
}
