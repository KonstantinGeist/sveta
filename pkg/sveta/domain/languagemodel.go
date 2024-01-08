package domain

type LanguageModel interface {
	// Complete completes the given prompt by using the underlying LLM (large language model).
	Complete(prompt string) (string, error)
}
