package domain

// Lists intrinsic config keys supported by the AI.

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
)
