package domain

// Lists built-in config keys supported by the AI.

const (
	// ConfigKeyAgentName the agent's name will be found in all prompts etc.
	ConfigKeyAgentName = "agentName"
	// ConfigKeyContext a short description of the AI agent which helps it understand how it should behave
	ConfigKeyContext = "context"
	// ConfigKeyLogPath file path where to save the logs
	ConfigKeyLogPath = "logPath"
	// ConfigKeyWorkingMemorySize the maximum number of latest dialog lines which should be considered as part of
	// the "context" when responding to a prompt
	ConfigKeyWorkingMemorySize = "workingMemorySize"
	// ConfigKeyWorkingMemoryMaxAge we don't want to recall working memory which is too old (it's most likely already irrelevant).
	// Specifies what's considered "old" memory, in milliseconds.
	ConfigKeyWorkingMemoryMaxAge = "workingMemoryMaxAge"
	// ConfigKeyEpisodicMemoryTopCount the amount of top results from recalled episodic memory to include in the context
	ConfigKeyEpisodicMemoryTopCount = "episodicMemoryTopCount"
	// ConfigKeyEpisodicMemorySurroundingCount the amount of surrounding memories to include in the context, relative
	// to the top results (see also ConfigKeyEpisodicMemoryTopCount)
	ConfigKeyEpisodicMemorySurroundingCount = "episodicMemorySurroundingCount"
	// ConfigKeyEpisodicMemorySimilarityThreshold what embedding similarity is considered so low we don't want to
	// include it in the context at all (even if it's the top result)
	ConfigKeyEpisodicMemorySimilarityThreshold = "episodicMemorySimilarityThreshold"
)
