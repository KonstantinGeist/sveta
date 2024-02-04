package domain

// A list of built-in config keys supported by the AI's core (i.e. settings of non-core AI filters are not included).

const (
	// ConfigKeyAgentName the agent's name will be found in all prompts etc.
	ConfigKeyAgentName = "agentName"
	// ConfigKeyAgentDescription a short description of the AI agent which helps it understand how it should behave
	ConfigKeyAgentDescription = "agentDescription"
	// ConfigKeyLogPath file path where to save the logs
	ConfigKeyLogPath = "logPath"
	// ConfigKeyWorkingMemorySize the maximum number of latest dialog lines which should be considered as part of
	// the "context" when responding to a prompt
	ConfigKeyWorkingMemorySize = "workingMemorySize"
	// ConfigKeyWorkingMemoryMaxAge we don't want to recall working memory which is too old (it's most likely already irrelevant).
	// Specifies what's considered "old" memory, in milliseconds.
	ConfigKeyWorkingMemoryMaxAge = "workingMemoryMaxAge"
	// ConfigKeyEpisodicMemoryFirstStageTopCount the amount of top results from recalled episodic memory to include in the context (first stage)
	ConfigKeyEpisodicMemoryFirstStageTopCount = "episodicMemoryFirstStageTopCount"
	// ConfigKeyEpisodicMemorySecondStageTopCount the amount of top results from recalled episodic memory to include in the context (second stage)
	ConfigKeyEpisodicMemorySecondStageTopCount = "episodicMemorySecondStageTopCount"
	// ConfigKeyEpisodicMemorySurroundingCount the amount of surrounding memories to include in the context, relative
	// to the top results (see also ConfigKeyEpisodicMemoryFirstStageTopCount)
	ConfigKeyEpisodicMemorySurroundingCount = "episodicMemorySurroundingCount"
	// ConfigKeyEpisodicMemorySimilarityThreshold what embedding similarity is considered so low we don't want to
	// include it in the context at all (even if it's the top result)
	ConfigKeyEpisodicMemorySimilarityThreshold = "episodicMemorySimilarityThreshold"
	// ConfigKeyRankerMaxMemorySize specifies the maximum size of a recalled memory when passed to  the reranker (to reduce the amount of data sent to it)
	ConfigKeyRankerMaxMemorySize = "rankerMaxMemorySize"
	// ConfigKeyResponseRetryCount how many times we should try retrieve an answer from an LLM in case it fails for some reason,
	// before we finally return an error.
	ConfigKeyResponseRetryCount = "responseRetryCount"
)
