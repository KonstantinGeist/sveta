package function

import (
	"fmt"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/rewrite"
)

type filter struct {
	functionService  *domain.FunctionService
	functionJobQueue *common.JobQueue
	logger           common.Logger
}

func NewFilter(
	functionService *domain.FunctionService,
	functionJobQueue *common.JobQueue,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		functionService:  functionService,
		functionJobQueue: functionJobQueue,
		logger:           logger,
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	inputMemory := context.MemoryCoalesced([]string{rewrite.DataKeyRewrittenInput, domain.DataKeyInput})
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	closures, err := f.functionService.CreateClosures(inputMemory.What)
	if err != nil {
		f.logger.Log("failed to create closures: " + err.Error())
		return nextAIFilterFunc(context)
	}
	if len(closures) > 0 {
		// We schedule the function asynchronously to
		// a) avoid running potentially heavyweight operations on the same goroutine
		// b) avoid recursive mutexes, since we maybe already under a mutex here who knows
		// what mutex the client code in the Body lambda locks
		f.functionJobQueue.Enqueue(func() error {
			for _, closure := range closures {
				err = closure.Invoke()
				if err != nil {
					f.logger.Log(fmt.Sprintf("failed to invoke closure \"%s\": %s", closure.Name, err.Error()))
				}
			}
			return nil
		})
		// If successful, stops the entire chain of AI filters for now.
		return nil
	}
	return nextAIFilterFunc(context)
}
