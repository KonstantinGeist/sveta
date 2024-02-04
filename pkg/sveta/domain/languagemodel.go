package domain

// LanguageModel a generic interface for a large language model (LLM).
type LanguageModel interface {
	// Name the name of the model. Useful for debugging.
	Name() string
	// Modes response modes supported by the model. Some models are not good at JSON or roleplay, so we want LanguageModelSelector
	// to take that into consideration.
	Modes() []ResponseMode
	// Complete completes the given prompt by using the underlying LLM (large language model). `jsonMode` makes sure
	// the output will be a syntactically valid JSON (grammar-restricted completion).
	Complete(prompt string, jsonMode bool) (string, error)
	// PromptFormatter the prompt formatter associated with this language model. Different language models assume
	// different formatting rules and can be quite sensitive to slight variations.
	PromptFormatter() PromptFormatter
}
