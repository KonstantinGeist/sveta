package domain

// LanguageModel a generic interface for a large language model (LLM).
type LanguageModel interface {
	// Name the name of the model. Useful for debugging.
	Name() string
	// ResponseModes response modes supported by the model. Some models are not good at JSON or roleplay, so we want LanguageModelSelector
	// to take that into consideration.
	ResponseModes() []ResponseMode
	// Complete completes the given prompt by using the underlying LLM (large language model).
	Complete(prompt string, options CompleteOptions) (string, error)
	// LegacyPromptFormatter the prompt formatter associated with this language model. Different language models assume
	// different formatting rules and can be quite sensitive to slight variations.
	LegacyPromptFormatter() LegacyPromptFormatter
	// PromptFormatter2 the prompt formatter associated with this language model. Different language models assume
	// different formatting rules and can be quite sensitive to slight variations.
	PromptFormatter2() PromptFormatter2
	// ResponseCleaner cleans the response by fixing known issues specific to the current model
	ResponseCleaner() ResponseCleaner
}
