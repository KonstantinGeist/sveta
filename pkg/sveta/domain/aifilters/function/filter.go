package function

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/aifilters/rewrite"
)

type filter struct {
	memoryFactory   domain.MemoryFactory
	functionService *domain.FunctionService
	logger          common.Logger
}

func NewFilter(
	memoryFactory domain.MemoryFactory,
	functionService *domain.FunctionService,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		memoryFactory:   memoryFactory,
		functionService: functionService,
		logger:          logger,
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
	if len(closures) == 0 { // if no closures are matched, just pass it through
		return nextAIFilterFunc(context)
	}
	var outputs []string
	var stops int
	for _, closure := range closures {
		output, err := closure.Invoke()
		if err != nil {
			f.logger.Log(fmt.Sprintf("failed to invoke closure \"%s\": %s", closure.Name, err.Error()))
			continue
		}
		if output.Output != "" {
			outputs = append(outputs, output.Output)
		}
		if output.Stop {
			stops++
		}
	}
	if stops > 0 {
		return nil
	}
	if len(outputs) == 0 {
		return nextAIFilterFunc(context) // if there are no outputs, just pass it through
	}
	// TODO internationalize
	output := fmt.Sprintf(
		"Additional information to use when answering: \"%s\" (use it if it's relevant to the question below).\n%s (respond in the style of your persona)",
		strings.Join(outputs, " "),
		inputMemory.What,
	)
	outputMemory := f.memoryFactory.NewMemory(domain.MemoryTypeDialog, inputMemory.Who, output, inputMemory.Where)
	return nextAIFilterFunc(context.
		WithMemory(rewrite.DataKeyRewrittenInput, outputMemory).
		WithMemory(domain.DataKeyInput, outputMemory),
	)
}
