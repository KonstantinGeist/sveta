package code

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/domain/passes/response"
)

// TODO get file paths from the config
// TODO use DI for launching in docker, for writing to file etc.
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
	aiContext           *domain.AIContext
	memoryFactory       domain.MemoryFactory
	codeResponseService *domain.ResponseService
	jsonResponseService *domain.ResponseService
	namedMutexAcquirer  domain.NamedMutexAcquirer
	logger              common.Logger
}

func NewPass(
	aiContext *domain.AIContext,
	memoryFactory domain.MemoryFactory,
	codeResponseService *domain.ResponseService,
	jsonResponseService *domain.ResponseService,
	namedMutexAcquirer domain.NamedMutexAcquirer,
	logger common.Logger,
) domain.Pass {
	return &pass{
		aiContext:           aiContext,
		memoryFactory:       memoryFactory,
		codeResponseService: codeResponseService,
		jsonResponseService: jsonResponseService,
		namedMutexAcquirer:  namedMutexAcquirer,
		logger:              logger,
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
		p.logger.Log("failed to generate Python code")
		return nextPassFunc(context)
	}
	if code == "" {
		return nextPassFunc(context)
	}
	err = p.preparePythonFile(code)
	if err != nil {
		p.logger.Log("failed to prepare Python file")
		return nextPassFunc(context)
	}
	result, err := p.runCodeInDocker()
	if err != nil {
		p.logger.Log("failed to run Python file")
		return nextPassFunc(context)
	}
	if result == "" {
		result = "done"
	}
	satisfies, err := p.satifies(input, result)
	if err != nil {
		p.logger.Log("failed to evaluate if the answer satisfies the question/task")
		return nextPassFunc(context)
	}
	if !satisfies {
		return nextPassFunc(context)
	}
	outputMemory := p.memoryFactory.NewMemory(domain.MemoryTypeDialog, p.aiContext.AgentName, result, inputMemory.Where)
	context.Data[response.DataKeyOutput] = outputMemory
	return nil
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
		fmt.Sprintf("Question or task: \"%s\".\nAnswer: \"%s\".\n\nDoes the answer satisfy the question/task? Return only yes or no.\n", input, result),
		&output,
	)
	if err != nil {
		return false, err
	}
	returnedValue := strings.ToLower(strings.TrimSpace(output.ReturnedValue))
	return returnedValue == "yes", nil
}

func (p *pass) getCodeResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext("CodeLLM", "You are an AI programming assistant, utilizing the DeepSeek Coder model, developed by DeepSeek Company, and you only answer by outputting Python code and nothing else.", "")
	return p.codeResponseService.WithAIContext(rankerAIContext).WithRetryCount(1)
}

func (p *pass) getEvaluatorResponseService() *domain.ResponseService {
	rankerAIContext := domain.NewAIContext("EvaluatorLLM", "You are EvaluatorLLM, an intelligent assistant which decides if the answer satisfies the question/task.", "")
	return p.jsonResponseService.WithAIContext(rankerAIContext)
}

func (p *pass) runCodeInDocker() (string, error) {
	namedMutex, err := p.namedMutexAcquirer.AcquireNamedMutex("codePassDocker", time.Minute)
	if err != nil {
		return "", err
	}
	defer namedMutex.Release()
	cmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s/sandbox:/usr/src/app", os.Getenv("PWD")), "python:3-alpine", "python", "/usr/src/app/code.py") // Create a pipe to capture the output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("error creating stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("error starting command: %w", err)
	}
	output, err := io.ReadAll(stdout)
	if err != nil {
		return "", fmt.Errorf("error reading stdout: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("command failed: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

func (p *pass) preparePythonFile(content string) error {
	file, err := os.Create("sandbox/code.py")
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	_, err = file.WriteString(content)
	if err != nil {
		return err
	}
	return nil
}
