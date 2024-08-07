package response

import (
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/rewrite"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/workingmemory"
)

const responseCapability = "response"
const episodicMemoryCapability = "episodicMemory"
const rerankCapability = "rerank"
const hydeCapability = "hyde"
const routerCapability = "router"

type pass struct {
	aiContext                         *domain.AIContext
	memoryFactory                     domain.MemoryFactory
	memoryRepository                  domain.MemoryRepository
	defaultResponseService            *domain.ResponseService
	rerankResponseService             *domain.ResponseService
	embedder                          domain.Embedder
	logger                            common.Logger
	episodicMemoryFirstStageTopCount  int
	episodicMemorySecondStageTopCount int
	episodicMemorySurroundingCount    int
	episodicMemorySimilarityThreshold float64
	rerankerMaxMemorySize             int
}

func NewPass(
	aiContext *domain.AIContext,
	memoryFactory domain.MemoryFactory,
	memoryRepository domain.MemoryRepository,
	defaultResponseService *domain.ResponseService,
	rerankResponseService *domain.ResponseService,
	embedder domain.Embedder,
	config *common.Config,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:                         aiContext,
		memoryFactory:                     memoryFactory,
		memoryRepository:                  memoryRepository,
		defaultResponseService:            defaultResponseService,
		rerankResponseService:             rerankResponseService,
		embedder:                          embedder,
		logger:                            logger,
		episodicMemoryFirstStageTopCount:  config.GetIntOrDefault(domain.ConfigKeyEpisodicMemoryFirstStageTopCount, 10),
		episodicMemorySecondStageTopCount: config.GetIntOrDefault(domain.ConfigKeyEpisodicMemorySecondStageTopCount, 3),
		episodicMemorySurroundingCount:    config.GetIntOrDefault(domain.ConfigKeyEpisodicMemorySurroundingCount, 1),
		episodicMemorySimilarityThreshold: config.GetFloatOrDefault(domain.ConfigKeyEpisodicMemorySimilarityThreshold, 0.1),
		rerankerMaxMemorySize:             config.GetIntOrDefault(domain.ConfigKeyRerankerMaxMemorySize, 500),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        responseCapability,
			Description: "generates a response to the user query",
		},
		{
			Name:        episodicMemoryCapability,
			Description: "enriches the user query with information recalled from the episodic memory",
		},
		{
			Name:        rerankCapability,
			Description: "reranks the recalled memory according to the relevance to the user query",
		},
		{
			Name:        hydeCapability,
			Description: "improves episodic memory recall by reformulating the user query in several different ways",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(responseCapability) {
		return nextPassFunc(context)
	}
	// ignore this pass if some previous pass already generated the output
	outputMemory := context.Memory(domain.DataKeyOutput)
	if outputMemory != nil {
		return nextPassFunc(context)
	}
	rewrittenInputMemory := context.MemoryCoalesced([]string{rewrite.DataKeyRewrittenInput, domain.DataKeyInput})
	if rewrittenInputMemory == nil {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	workingMemories := context.Memories(workingmemory.DataKeyWorkingMemory)
	episodicMemories, err := p.recallFromEpisodicMemory(context, workingMemories, inputMemory, rewrittenInputMemory)
	if err != nil {
		return err
	}
	memories := domain.MergeMemories(episodicMemories, workingMemories...)
	memories = domain.MergeMemories(memories, inputMemory)
	response, err := p.defaultResponseService.RespondToMemoriesWithText(memories, domain.ResponseModeNormal)
	if err != nil {
		return err
	}
	responseMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, response, inputMemory.Where)
	return nextPassFunc(context.WithMemory(domain.DataKeyOutput, responseMemory))
}

// recallFromEpisodicMemory finds memories in the so-called "episodic memory", or long-term memory which may contain memories from long ago
func (p *pass) recallFromEpisodicMemory(context *domain.PassContext, workingMemories []*domain.Memory, inputMemory, rewrittenInputMemory *domain.Memory) ([]*domain.Memory, error) {
	if !context.IsCapabilityEnabled(responseCapability) {
		return nil, nil
	}
	var embeddingsToSearch []domain.Embedding
	inputEmbeddingsToSearch := p.getHypotheticalEmbeddings(context, inputMemory)
	rewrittenEmbeddingsToSearch := p.getHypotheticalEmbeddings(context, rewrittenInputMemory)
	embeddingsToSearch = append(embeddingsToSearch, inputEmbeddingsToSearch...)
	embeddingsToSearch = append(embeddingsToSearch, rewrittenEmbeddingsToSearch...)
	if inputMemory.Embedding != nil {
		embeddingsToSearch = append(embeddingsToSearch, *inputMemory.Embedding)
	}
	if rewrittenInputMemory.Embedding != nil {
		embeddingsToSearch = append(embeddingsToSearch, *rewrittenInputMemory.Embedding)
	}
	episodicMemories, err := p.memoryRepository.FindByEmbeddings(domain.EmbeddingFilter{
		Where:               rewrittenInputMemory.Where,
		Embeddings:          embeddingsToSearch,
		TopCount:            p.episodicMemoryFirstStageTopCount,
		SurroundingCount:    p.episodicMemorySurroundingCount,
		ExcludedIDs:         domain.GetMemoryIDs(workingMemories), // don't recall what's already in the input
		SimilarityThreshold: p.episodicMemorySimilarityThreshold,
	})
	if err != nil {
		return nil, err
	}
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	episodicMemories = p.rankMemoriesAndGetTopN(context, episodicMemories, rewrittenInputMemory.What, rewrittenInputMemory.Where)
	if len(episodicMemories) == 0 {
		return nil, nil
	}
	p.logRecalledMemories(episodicMemories)
	return episodicMemories, nil
}

func (p *pass) getEmbedding(what string) *domain.Embedding {
	embedding, err := p.embedder.Embed(what)
	if err != nil {
		p.logger.Log(err.Error())
		return nil
	}
	return &embedding
}

func (p *pass) logRecalledMemories(memories []*domain.Memory) {
	memories = domain.FilterMemoriesByTypes(memories, []domain.MemoryType{domain.MemoryTypeDialog})
	var builder strings.Builder
	for _, memory := range memories {
		builder.WriteString(memory.Who)
		builder.WriteString(": ")
		builder.WriteString(memory.What)
		builder.WriteString("\n")
	}
	p.logger.Log(fmt.Sprintf("\n======\nRecalled context:\n%s\n========\n", builder.String()))
}
