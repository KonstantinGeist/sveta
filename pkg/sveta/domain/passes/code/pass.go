package code

import (
	"errors"
	"fmt"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO get file paths from the config
// TODO try input and rewritteInpit one by one if rewrittenInput is not satisfied?
// TODO files not created for some reason
// TODO maybe always prepend "here is the output: " to make evaluator pass
// TODO are Sveta's responses saved to memory?
// TODO add timeout for processes (10 seconds or smth)
// TODO memory messes it up?
// TODO use Solar as evaluator
// TODO reject outputs if conflicts with persona (maybe into evaluator)

const codeCapability = "code"

type pass struct {
	aiContext             *domain.AIContext
	memoryFactory         domain.MemoryFactory
	summaryRepository     domain.SummaryRepository
	codeResponseService   *domain.ResponseService
	jsonResponseService   *domain.ResponseService
	normalResponseService *domain.ResponseService
	runner                Runner
	logger                common.Logger
}

func NewPass(
	aiContext *domain.AIContext,
	memoryFactory domain.MemoryFactory,
	summaryRepository domain.SummaryRepository,
	codeResponseService *domain.ResponseService,
	jsonResponseService *domain.ResponseService,
	normalResponseService *domain.ResponseService,
	runner Runner,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:             aiContext,
		memoryFactory:         memoryFactory,
		summaryRepository:     summaryRepository,
		codeResponseService:   codeResponseService,
		jsonResponseService:   jsonResponseService,
		normalResponseService: normalResponseService,
		runner:                runner,
		logger:                logger,
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        codeCapability,
			Description: "interprets code",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(codeCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	input := inputMemory.What
	code, err := p.generateCode(input)
	if err != nil && !errors.Is(err, domain.ErrFailedToResponse) {
		p.logger.Log("failed to generate Python code: " + err.Error())
		return nextPassFunc(context)
	}
	if code == "" {
		p.logger.Log("CODE refused to answer\n")
		return nextPassFunc(context)
	}
	result, err := p.runner.Run(code)
	if err != nil {
		p.logger.Log("failed to run code: " + err.Error())
		return nextPassFunc(context)
	}
	if result == "" {
		result = "done"
	}
	satisfies, err := p.satifies(input, result)
	if err != nil {
		p.logger.Log("failed to evaluate if the answer satisfies the question/task: " + err.Error())
		return nextPassFunc(context)
	}
	if !satisfies {
		return nextPassFunc(context)
	}
	reformulatedResult, err := p.reformulate(input, result, inputMemory.Where)
	if err != nil {
		p.logger.Log("failed to reformulate the answer: " + err.Error())
	} else {
		if reformulatedResult != "" {
			satisfies, err = p.satifies(input, reformulatedResult)
			if err != nil {
				p.logger.Log("failed to evaluate if the answer satisfies the question/task: " + err.Error())
			} else {
				result = reformulatedResult
			}
		}
	}
	outputMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, result, inputMemory.Where)
	context.Data[domain.DataKeyOutput] = outputMemory
	return nextPassFunc(context)
}

func (p *pass) generateCode(input string) (string, error) {
	query := fmt.Sprintf("Problem: \"%s\". Output Python code which solves the problem and nothing else. If the problem cannot be solved by running Python code, refuse to answer. The generated code should print its result to the output. If the request is not an explicit command to process text or files, refuse to answer.", input)
	queryMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, "User", query, "")
	return p.getCodeResponseService().RespondToMemoriesWithText([]*domain.Memory{queryMemory}, domain.ResponseModeCode)
}

func (p *pass) satifies(input, result string) (bool, error) {
	var output struct {
		Reasoning     string `json:"reasoning"`
		ReturnedValue string `json:"returnedValue"`
	}
	err := p.getEvaluatorResponseService().RespondToQueryWithJSON(
		fmt.Sprintf("Question or task: \"%s\".\nAnswer: \"%s\".\n\nDoes the answer appear to satisfy the question/task? Provide the reasoning and return only yes or no. Answer yes even if the answer is not entirely accurate.\n", input, result),
		&output,
	)
	if err != nil {
		return false, err
	}
	p.logger.Log(fmt.Sprintf("\n\nCODE input: \"%s\", result: \"%s\", reasoning: \"%s\", returnedValue: \"%s\"\n\n", input, result, output.Reasoning, output.ReturnedValue))
	returnedValue := strings.ToLower(strings.TrimSpace(output.ReturnedValue))
	return returnedValue == "yes", nil
}

func (p *pass) reformulate(input, output, where string) (string, error) {
	summary, err := p.summaryRepository.FindByWhere(where)
	if err != nil {
		return "", err
	}
	if summary == nil {
		defaultSummary := "no summary"
		summary = &defaultSummary
	}
	what := fmt.Sprintf("Chat summary: \"%s\". Persona: \"%s\". Question or task: \"%s\". Answer: \"%s\". Reformulate the answer in accordance with the provided persona and the chat summary. Output only the reformulated answer and nothing else. The reformulated answer must preserve the original meaning/answer. Pay most attention to the user's LAST question/task.", *summary, p.aiContext.AgentDescription, input, output)
	memoryToReformulate := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, what, where)
	reformulated, err := p.getPersonaResponseService().RespondToMemoriesWithText([]*domain.Memory{memoryToReformulate}, domain.ResponseModeNormal)
	if err != nil {
		return "", err
	}
	reformulated = strings.TrimSpace(reformulated)
	if len(reformulated) > 2 && reformulated[0] == '"' && reformulated[len(reformulated)-1] == '"' {
		reformulated = reformulated[1 : len(reformulated)-2]
	}
	return strings.TrimSpace(reformulated), nil
}

func (p *pass) getCodeResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext("CodeLLM", "You are an AI programming assistant, utilizing the DeepSeek Coder model, developed by DeepSeek Company, and you only answer by outputting Python code and nothing else.", "")
	return p.codeResponseService.WithAIContext(rankerAIContext).WithRetryCount(1)
}

func (p *pass) getEvaluatorResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext("EvaluatorLLM", "You are EvaluatorLLM, an intelligent assistant which decides if the answer satisfies the question/task.", "")
	return p.jsonResponseService.WithAIContext(rankerAIContext)
}

func (p *pass) getPersonaResponseService() *domain.ResponseService {
	personaAIContext := domain.NewAIContext("PersonaLLM", "You are PersonaLLM, an intelligent assistant which reformulates text based on a given persona.", "")
	return p.jsonResponseService.WithAIContext(personaAIContext)
}
