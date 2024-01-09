package llama

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

var errUnexpectedModelOutput = errors.New("unexpected model output")

const (
	ConfigKeyLLMTemperature    = "llmTemperature"
	ConfigKeyLLMContextSize    = "llmContextSize"
	ConfigKeyLLMCPUThreadCount = "llmCPUThreadCount"
	ConfigKeyLLMGPULayerCount  = "llmGPULayerCount"
	ConfigKeyLLMGRepeatPenalty = "llmRepeatPenalty"
)

type languageModel struct {
	mutex          sync.Mutex
	httpClient     http.Client
	logger         common.Logger
	agentName      string
	inferCommand   string
	temperature    float64
	contextSize    int
	gpuLayerCount  int
	cpuThreadCount int
	repeatPenalty  float64
}

func NewLanguageModel(agentName string, config *common.Config) domain.LanguageModel {
	return &languageModel{
		agentName:      agentName,
		temperature:    config.GetFloatOrDefault(ConfigKeyLLMTemperature, 0.7),
		contextSize:    config.GetIntOrDefault(ConfigKeyLLMContextSize, 4096),
		gpuLayerCount:  config.GetIntOrDefault(ConfigKeyLLMGPULayerCount, 40),
		cpuThreadCount: config.GetIntOrDefault(ConfigKeyLLMCPUThreadCount, 6),
		repeatPenalty:  config.GetFloatOrDefault(ConfigKeyLLMGRepeatPenalty, 1.1),
	}
}

func (l *languageModel) Complete(prompt string) (string, error) {
	// Only 1 request can be processed at a time currently because we run Sveta on commodity hardware which can't
	// usually process two requests simultaneously due to low amounts of VRAM.
	l.mutex.Lock()
	defer l.mutex.Unlock()
	command, err := l.buildInferCommand()
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	// We keep track of how many times the agent name with the delimiter was found in the output, to understand
	// when we should stop token generation because otherwise the model tries to continue the dialog forever, and we want
	// to stop as soon as possible. Note that further the caller will remove unnecessary continuations, too.
	agentNameCount := strings.Count(prompt, l.getAgentNameWithDelimiter())
	newAgentNamePromptCount := 0
	err = runInferCommand(command, prompt, func(s string) bool {
		// See the comment to `agentNameCount` variable definition.
		if strings.Contains(s, l.getAgentNameWithDelimiter()) {
			newAgentNamePromptCount++
			if newAgentNamePromptCount > agentNameCount {
				return false
			}
		}
		buf.WriteString(s)
		return true
	})
	if err != nil {
		// A process can run successfully but be terminated with a SIGKILL for some reason.
		// So we ignore it but log it, leaving what has been generated so far intact.
		_, ok := err.(*exec.ExitError)
		if !ok {
			fmt.Println(err)
		}
	}
	output := buf.String()
	if len(output) < len(prompt)+1 {
		return "", errUnexpectedModelOutput
	}
	// The model repeats what was said before, so we remove it from the response.
	return strings.TrimSpace(output[len(prompt)+1:]), nil
}

func (l *languageModel) buildInferCommand() (string, error) {
	if l.inferCommand != "" {
		return l.inferCommand, nil
	}
	shellTemplate := "%s/infer -m %s/model.bin "
	shellTemplate += fmt.Sprintf(
		"-t %d -ngl %d --color -c %d --temp %f --repeat_penalty %f -n -1 -p ",
		l.cpuThreadCount,
		l.gpuLayerCount,
		l.contextSize,
		l.temperature,
		l.repeatPenalty,
	)
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	l.inferCommand = fmt.Sprintf(shellTemplate, workingDirectory, workingDirectory)
	return l.inferCommand, nil
}

func (l *languageModel) getAgentNameWithDelimiter() string {
	return l.agentName + ":"
}

// We hook up to the llama.cpp binary by launching a subprocess and reading its standard output until
// processLineFunc(..) signals it should stop with false as the returned value.
func runInferCommand(cmdstr, prompt string, processLineFunc func(s string) bool) error {
	args := strings.Fields(cmdstr)
	args = append(args, prompt)
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	defer cancelFunc()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...) // TODO shell injections?
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var wg sync.WaitGroup
	wg.Add(1)
	scanner := bufio.NewScanner(stdout)
	go func() {
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			keepRunning := processLineFunc(line)
			if !keepRunning {
				cancelFunc() // the process function signals we should stop because a certain condition has been met
				break
			}
		}
		wg.Done()
	}()
	if err = cmd.Start(); err != nil {
		return err
	}
	wg.Wait()
	return cmd.Wait()
}
