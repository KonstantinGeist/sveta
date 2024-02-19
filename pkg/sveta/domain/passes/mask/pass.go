package mask

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const maskCapability = "mask"

const maskThreshold = 20

type pass struct {
	responseService *domain.ResponseService
	memoryFactory   domain.MemoryFactory
	logger          common.Logger
}

func NewPass(
	responseService *domain.ResponseService,
	memoryFactory domain.MemoryFactory,
	logger common.Logger,
) domain.Pass {
	return &pass{
		responseService: responseService,
		memoryFactory:   memoryFactory,
		logger:          logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        maskCapability,
			Description: "determines which passes need to run",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	maskableCapabilities := p.getMaskableCapabilities(context)
	if len(maskableCapabilities) == 0 {
		return nextPassFunc(context)
	}
	query := p.formatQuery(maskableCapabilities, inputMemory)
	type tool struct {
		ToolName       string `json:"toolName"`
		ScoreInPercent int    `json:"scoreInPercent"`
	}
	var output struct {
		Tools []tool `json:"tools"`
	}
	output.Tools = append(output.Tools, tool{})
	err := p.getMaskResponseService().RespondToQueryWithJSON(query, &output)
	if err != nil {
		p.logger.Log("failed to mask passes: " + err.Error())
		return nextPassFunc(context)
	}
	var tools []string
	for _, outputTool := range output.Tools {
		if outputTool.ScoreInPercent > maskThreshold {
			tools = append(tools, outputTool.ToolName)
		}
	}
	capabilities := p.getCapabilities(context.EnabledCapabilities, tools)
	if len(capabilities) == 0 { // can't be right => wrong output => assume all capabilities
		return nextPassFunc(context)
	}
	p.logCapabilities(capabilities)
	unmaskableCapabalities := p.getUnmaskableCapabilities(context)
	capabilities = append(capabilities, unmaskableCapabalities...)
	return nextPassFunc(context.WithCapabilities(capabilities))
}

func (p *pass) getMaskResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext(
		"ToolLLM",
		"You're ToolLLM, an intelligent assistant that decides which text processing tools can solve the user query.",
		"",
	)
	return p.responseService.WithAIContext(rankerAIContext)
}

func (p *pass) getCapabilities(capabilities []*domain.Capability, names []string) []*domain.Capability {
	var result []*domain.Capability
	for _, name := range names {
		name = strings.TrimSpace(name)
		var foundCapability *domain.Capability
		for _, capability := range capabilities {
			if capability.Name == name {
				foundCapability = capability
				break
			}
		}
		if foundCapability != nil {
			result = append(result, foundCapability)
		}
	}
	return result
}

func (p *pass) getMaskableCapabilities(context *domain.PassContext) []*domain.Capability {
	maskableCapabilities := make([]*domain.Capability, 0, len(context.EnabledCapabilities))
	for _, capability := range context.EnabledCapabilities {
		if capability.IsMaskable {
			maskableCapabilities = append(maskableCapabilities, capability)
		}
	}
	return maskableCapabilities
}

func (p *pass) getUnmaskableCapabilities(context *domain.PassContext) []*domain.Capability {
	maskableCapabilities := make([]*domain.Capability, 0, len(context.EnabledCapabilities))
	for _, capability := range context.EnabledCapabilities {
		if !capability.IsMaskable {
			maskableCapabilities = append(maskableCapabilities, capability)
		}
	}
	return maskableCapabilities
}

func (p *pass) formatQuery(maskableCapabilities []*domain.Capability, inputMemory *domain.Memory) string {
	var builder strings.Builder
	builder.WriteString("The following is a list of available text processing tools:\n\n")
	for index, capability := range maskableCapabilities {
		builder.WriteString(fmt.Sprintf("%d) %s (%s)\n", index+1, capability.Name, capability.Description))
	}
	builder.WriteString(
		fmt.Sprintf(
			"Score how much the given tools can solve the user query, from 1%% to 100%%: \"%s\". Score ALL tools.\n",
			inputMemory.What))
	return builder.String()
}

func (p *pass) logCapabilities(capabilities []*domain.Capability) {
	var builder strings.Builder
	builder.WriteString("[CHOSEN CAPABILITIES]\n")
	for _, capability := range capabilities {
		builder.WriteString(capability.Name)
		builder.WriteString("\n")
	}
	builder.WriteString("[END]\n")
	p.logger.Log(builder.String())
	fmt.Println(builder.String())
}
