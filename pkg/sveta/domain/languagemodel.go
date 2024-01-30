package domain

type LanguageModelPurpose int

const (
	LanguageModelPurposeGeneric = LanguageModelPurpose(iota)
	LanguageModelPurposeJSON
	LanguageModelPurposeRolePlay
)

type LanguageModel interface {
	Name() string
	Purpose() LanguageModelPurpose
	// Complete completes the given prompt by using the underlying LLM (large language model).
	Complete(prompt string, jsonMode bool) (string, error)
	PromptFormatter() PromptFormatter
}
