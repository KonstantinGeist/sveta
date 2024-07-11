package domain

// ResponseMode different language models are used for different purposes in Sveta.
type ResponseMode int

const (
	// ResponseModeNormal the default response mode
	ResponseModeNormal = ResponseMode(iota)
	// ResponseModeJSON the model is good for JSON output
	ResponseModeJSON
	// ResponseModeRerank the model is good for reranking
	ResponseModeRerank
	// ResponseModeCode the model is good for writing code
	ResponseModeCode
)

var ResponseModes = []ResponseMode{
	ResponseModeJSON,
	ResponseModeNormal,
	ResponseModeRerank,
}
