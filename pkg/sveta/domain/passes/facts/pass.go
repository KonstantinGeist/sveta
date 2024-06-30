package facts

import (
	"fmt"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/workingmemory"
)

const factsCapability = "facts"

type pass struct {
	aiContext             *domain.AIContext
	memoryRepository      domain.MemoryRepository
	memoryFactory         domain.MemoryFactory
	responseService       *domain.ResponseService
	languageModelJobQueue *common.JobQueue
	logger                common.Logger
}

func NewPass(
	aiContext *domain.AIContext,
	memoryRepository domain.MemoryRepository,
	memoryFactory domain.MemoryFactory,
	responseService *domain.ResponseService,
	languageModelJobQueue *common.JobQueue,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:             aiContext,
		memoryRepository:      memoryRepository,
		memoryFactory:         memoryFactory,
		responseService:       responseService,
		languageModelJobQueue: languageModelJobQueue,
		logger:                logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        factsCapability,
			Description: "remembers facts",
			IsMaskable:  false,
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(factsCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	outputMemory := context.Memory(domain.DataKeyOutput)
	if inputMemory == nil || outputMemory == nil {
		return nextPassFunc(context)
	}
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	formattedMemories := p.formatMemories(domain.MergeMemories(workingMemories, []*domain.Memory{inputMemory, outputMemory}...))
	p.languageModelJobQueue.Enqueue(func() error {
		var output struct {
			Fact1 string `json:"fact1"`
			Fact2 string `json:"fact2"`
		}
		err := p.getSummarizerResponseService().RespondToQueryWithJSON(
			fmt.Sprintf("%s\nExtract facts from the chat history above into 2 short summaries at most (if possible). Example: \"User likes cat.\".", formattedMemories),
			&output,
		)
		if err != nil {
			return err
		}
		var facts []string
		if output.Fact1 != "" {
			facts = append(facts, output.Fact1)
		}
		if output.Fact2 != "" {
			facts = append(facts, output.Fact2)
		}
		for _, fact := range facts {
			existingMemory, err := p.memoryRepository.Find(domain.MemoryFilter{
				What:  fact,
				Where: inputMemory.Where,
			})
			if err != nil {
				p.logger.Log("failed to extract facts: " + err.Error())
				continue
			}
			if existingMemory != nil {
				continue
			}
			factMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, fact, inputMemory.Where)
			factMemory.When = time.Time{}
			err = p.memoryRepository.Store(factMemory)
			if err != nil {
				p.logger.Log("failed to extract facts: " + err.Error())
				continue
			}
		}
		return nil
	})
	return nextPassFunc(context)
}

func (p *pass) getSummarizerResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext(
		"FactLLM",
		"You're FactLLM, an intelligent assistant that extracts facts from the provided chat history."+
			"When extracting facts, pick the most relevant topics. "+
			"Example: \"User likes playing piano.\", etc.",
		"",
	)
	return p.responseService.WithAIContext(rankerAIContext)
}

func (p *pass) formatMemories(memories []*domain.Memory) string {
	var buf strings.Builder
	buf.WriteString("Chat history: ```\n")
	for _, memory := range memories {
		buf.WriteString(fmt.Sprintf("%s: %s\n\n", memory.Who, memory.What))
	}
	buf.WriteString("```\n\n")
	return buf.String()
}
