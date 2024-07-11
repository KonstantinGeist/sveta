package deepseekcoder

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO a mutex
// TODO get file paths from the config
// TODO add docker dependency to README

const maxCodeOutput = 1024

func RegisterCodeFunction(sveta api.API) error {
	return sveta.RegisterFunction(api.FunctionDesc{
		Name:        "runPythonCode",
		Description: "uses another LLM to interpret complex user queries using Python code interpreter (if it's an interpretable problem)",
		Parameters:  []domain.FunctionParameterDesc{},
		Body: func(input *api.FunctionInput) (api.FunctionOutput, error) {
			prompt := input.Input
			if prompt == "" {
				return api.FunctionOutput{}, nil
			}
			const agentName = "InterpretLLM"
			const agentDescription = "You are an AI programming assistant, utilizing the DeepSeek Coder model, developed by DeepSeek Company, and you only answer by outputting Python code and nothing else."
			const userName = "User"
			fullPrompt := fmt.Sprintf("Problem: \"%s\". Output Python code which solves the problem and nothing else. If the problem cannot be solved by running Python code, refuse to answer. The generated code should print its result to the output.", prompt)
			responseService := input.ResponseService.WithAIContext(domain.NewAIContext(agentName, agentDescription, ""))
			instructionMemory := domain.NewMemory("", domain.MemoryTypeDialog, userName, time.Now(), fullPrompt, "", nil)
			response, err := responseService.RespondToMemoriesWithText([]*domain.Memory{instructionMemory}, domain.ResponseModeCode)
			if err != nil {
				return api.FunctionOutput{}, err
			}
			const pythonCodeStart = "```python"
			const pythonCodeStop = "```"
			pythonCodeStartIndex := strings.Index(response, pythonCodeStart)
			if pythonCodeStartIndex == -1 {
				return api.FunctionOutput{}, errors.New("not python code start")
			}
			response = response[pythonCodeStartIndex+len(pythonCodeStart):]
			pythonCodeEndIndex := strings.Index(response, pythonCodeStop)
			pythonCode := strings.TrimSpace(response[0:pythonCodeEndIndex])
			err = preparePythonFile(pythonCode)
			if err != nil {
				return api.FunctionOutput{}, err
			}
			result, err := runCodeInDocker()
			if err != nil {
				return api.FunctionOutput{}, err
			}
			// trim output if too large because it can be abused
			if len(result) > maxCodeOutput {
				result = result[0:maxCodeOutput] + " <..>"
			}
			return domain.FunctionOutput{
				Output: fmt.Sprintf("Use MUST use this answer: \"%s\" to answer the following question or task: ", result),
			}, nil
		},
	})
}

func runCodeInDocker() (string, error) {
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
	return string(output), nil
}

func preparePythonFile(content string) error {
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
