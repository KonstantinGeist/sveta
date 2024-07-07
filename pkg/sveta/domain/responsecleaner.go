package domain

type CleanOptions struct {
	Prompt    string
	Response  string
	AgentName string
	Memories  []*Memory
}

// ResponseCleaner Sometimes, the model can generate too much (for example, trying to complete other participants' dialogs), so we trim it.
// The cleaner can also have additional post-processing specific to the model.
type ResponseCleaner interface {
	CleanResponse(options CleanOptions) string
}
