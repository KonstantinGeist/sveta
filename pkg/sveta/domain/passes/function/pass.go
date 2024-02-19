package function

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/rewrite"
)

type pass struct {
	memoryFactory   domain.MemoryFactory
	functionService *domain.FunctionService
	logger          common.Logger
}

func NewPass(
	memoryFactory domain.MemoryFactory,
	functionService *domain.FunctionService,
	logger common.Logger,
) domain.Pass {
	return &pass{
		memoryFactory:   memoryFactory,
		functionService: functionService,
		logger:          logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	functionDescs := p.functionService.FunctionDescs()
	result := make([]*domain.Capability, 0, len(functionDescs))
	for _, functionDesc := range functionDescs {
		result = append(result, &domain.Capability{
			Name:        functionDesc.Name,
			Description: functionDesc.Description,
			IsMaskable:  true,
		})
	}
	return result
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !p.areCapabilitiesEnabled(context) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	rewrittenInputMemory := context.Memory(rewrite.DataKeyRewrittenInput)
	input := p.getInput(inputMemory, rewrittenInputMemory)
	if input == "" {
		return nextPassFunc(context)
	}
	closures, err := p.functionService.CreateClosures(input)
	if err != nil {
		p.logger.Log("failed to create closures: " + err.Error())
		return nextPassFunc(context)
	}
	if len(closures) == 0 { // if no closures are matched, just pass it through
		return nextPassFunc(context)
	}
	var outputs []string
	var stops int
	for _, closure := range closures {
		output, err := closure.Invoke()
		if err != nil {
			p.logger.Log(fmt.Sprintf("failed to invoke closure \"%s\": %s", closure.Name, err.Error()))
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
		return nextPassFunc(context) // if there are no outputs, just pass it through
	}
	output := fmt.Sprintf(
		"Additional information to use when answering: \"%s\" (use it if it's relevant to the question below).\n%s (respond in the style of your persona)",
		strings.Join(outputs, " "),
		inputMemory.What,
	)
	outputMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, inputMemory.Who, output, inputMemory.Where)
	return nextPassFunc(context.
		WithMemory(rewrite.DataKeyRewrittenInput, outputMemory).
		WithMemory(domain.DataKeyInput, outputMemory),
	)
}

func (p *pass) areCapabilitiesEnabled(context *domain.PassContext) bool {
	for _, functionDesc := range p.functionService.FunctionDescs() {
		for _, enableCapability := range context.EnabledCapabilities {
			if functionDesc.Name == enableCapability.Name {
				return true
			}
		}
	}
	return false
}

func (p *pass) getInput(inputMemory, rewrittenInputMemory *domain.Memory) string {
	var input strings.Builder
	if rewrittenInputMemory != nil {
		input.WriteString(rewrittenInputMemory.What)
	}
	if inputMemory != nil {
		if rewrittenInputMemory != nil {
			input.WriteString(" In other words, ")
		}
		input.WriteString(inputMemory.What)
	}
	return input.String()
}
