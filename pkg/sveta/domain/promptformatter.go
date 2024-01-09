package domain

type PromptFormatter interface {
	// FormatDialog formats the given memory into a prompt which is best-suited for the underlying LLM (large language model).
	// Example:
	// 		Sveta:
	//      Hello
	//      John:
	//      Hello, too!
	FormatDialog(memories []*Memory) string
}
