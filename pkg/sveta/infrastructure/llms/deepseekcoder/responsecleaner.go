package deepseekcoder

import (
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type responseCleaner struct{}

func newResponseCleaner() *responseCleaner {
	return &responseCleaner{}
}

func (r *responseCleaner) CleanResponse(options domain.CleanOptions) string {
	return options.Response
}
