package domain

type PromptFormatter interface {
	// FormatDialog formats the given memory into a prompt which is best-suited for the underlying LLM (large language model).
	FormatDialog(memories []*Memory) string
}
