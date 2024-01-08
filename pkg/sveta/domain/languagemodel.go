package domain

type LanguageModel interface {
	Complete(prompt string) (string, error)
}
