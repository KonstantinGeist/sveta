package domain

import "time"

type PromptFormatter interface {
	// FormatDialog formats the given memory into a prompt which is best suited for the underlying LLM (large language model).
	// Example:
	//		### Sveta:
	//		Hello
	//		### John:
	//		Hello, too!
	FormatDialog(memories []*Memory) string

	// FormatAnnouncedTime formats the given time into a natural language string which announces the current time in
	// a format best suited for the current model.
	FormatAnnouncedTime(t time.Time) string

	// FormatJSONRequest formats the given query into a natural language string which requests the output to be
	// in JSON format.
	// Example: "Answer using JSON using the following JSON schema: %s"
	FormatJSONRequest(jsonSchemaQuery string) string
}
