package llamacpp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

var errUnexpectedModelOutput = errors.New("unexpected model output")

var mutex sync.Mutex

const (
	// ConfigKeyLLMDefaultTemperature how creative the output is
	ConfigKeyLLMDefaultTemperature = "llmDefaultTemperature"
	// ConfigKeyLLMContextSize the size of the context
	ConfigKeyLLMContextSize = "llmContextSize"
	// ConfigKeyLLMCPUThreadCount the number of CPUs used during inference
	ConfigKeyLLMCPUThreadCount = "llmCPUThreadCount"
	// ConfigKeyLLMGPULayerCount how many layers in the model can be offloaded to GPU
	ConfigKeyLLMGPULayerCount = "llmGPULayerCount"
	// ConfigKeyLLMGRepeatPenalty a coefficient against repetitions of same tokens
	ConfigKeyLLMGRepeatPenalty = "llmRepeatPenalty"
	// ConfigKeyLLResponseTimeout when to stop if the model takes too long to process input/generate output
	ConfigKeyLLResponseTimeout = "llmResponseTimeout"
)

type LanguageModel struct {
	logger                 common.Logger
	name                   string
	binPath                string
	responseModes          []domain.ResponseMode
	promptFormatter        domain.PromptFormatter
	agentNameWithDelimiter string
	defaultTemperature     float64
	contextSize            int
	gpuLayerCount          int
	cpuThreadCount         int
	repeatPenalty          float64
	responseTimeout        time.Duration
}

func (l *LanguageModel) Name() string {
	return l.name
}

// NewLanguageModel Creates a language model as implemented by llama.cpp
// `binPath` specifies the path to the target model relative to the bin folder (llama.cpp supports many models: Llama 2, Mistral, etc.)
// `config` contains parameters specific to the current GPU (see the constant above)
func NewLanguageModel(
	aiContext *domain.AIContext,
	modelName,
	binPath string,
	responseModes []domain.ResponseMode,
	promptFormatter domain.PromptFormatter,
	config *common.Config,
	logger common.Logger,
) *LanguageModel {
	return &LanguageModel{
		name:                   modelName,
		binPath:                binPath,
		responseModes:          responseModes,
		promptFormatter:        promptFormatter,
		agentNameWithDelimiter: getAgentNameWithDelimiter(aiContext, promptFormatter),
		logger:                 logger,
		defaultTemperature:     config.GetFloatOrDefault(ConfigKeyLLMDefaultTemperature, 0.7),
		contextSize:            config.GetIntOrDefault(ConfigKeyLLMContextSize, 4096),
		gpuLayerCount:          config.GetIntOrDefault(ConfigKeyLLMGPULayerCount, 40),
		cpuThreadCount:         config.GetIntOrDefault(ConfigKeyLLMCPUThreadCount, 6),
		repeatPenalty:          config.GetFloatOrDefault(ConfigKeyLLMGRepeatPenalty, 1.1),
		responseTimeout:        config.GetDurationOrDefault(ConfigKeyLLResponseTimeout, time.Minute),
	}
}

func (l *LanguageModel) ResponseModes() []domain.ResponseMode {
	return l.responseModes
}

func (l *LanguageModel) Complete(prompt string, options domain.CompleteOptions) (string, error) {
	// Only 1 request can be processed at a time currently because we run Sveta on commodity hardware which can't
	// usually process two requests simultaneously due to low amounts of VRAM.
	mutex.Lock()
	defer mutex.Unlock()
	command, err := l.buildInferCommand(options)
	if err != nil {
		return "", err
	}
	var buf strings.Builder
	// We keep track of how many times the agent name with the delimiter was found in the output, to understand
	// when we should stop token generation because otherwise the model can continue the dialog forever, and we want
	// to stop as soon as possible. Note that the caller will remove unnecessary continuations further, too.
	agentNameCount := strings.Count(prompt, l.agentNameWithDelimiter)
	newAgentNamePromptCount := 0
	err = runInferCommand(command, prompt, l.responseTimeout, func(s string) bool {
		// See the comment to `agentNameCount` variable definition.
		if strings.Contains(s, l.agentNameWithDelimiter) {
			newAgentNamePromptCount++
			if newAgentNamePromptCount > agentNameCount {
				return false
			}
		}
		buf.WriteString(s)
		return true
	})
	if err != nil {
		// A process can run successfully but be terminated with a SIGKILL for some reason (due to context cancellation?)
		// So we ignore it but log it, leaving what has been generated so far intact.
		_, ok := err.(*exec.ExitError)
		if !ok {
			l.logger.Log(err.Error())
		}
	}
	output := buf.String()
	if len(output) < len(prompt)+1 {
		return "", errUnexpectedModelOutput
	}
	// The model repeats what was said before, so we remove it from the response.
	return strings.TrimSpace(output[len(prompt)+1:]), nil
}

func (l *LanguageModel) PromptFormatter() domain.PromptFormatter {
	return l.promptFormatter
}

func (l *LanguageModel) buildInferCommand(options domain.CompleteOptions) (string, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return "", err
	}
	result := fmt.Sprintf("%s/llama.cpp", workingDirectory)
	if options.JSONMode {
		result += fmt.Sprintf(" --grammar-file %s/json.gbnf", workingDirectory)
	}
	result += fmt.Sprintf(" -m %s/", workingDirectory)
	result += fmt.Sprintf(
		"%s -t %d -ngl %d --color -c %d --temp %f --repeat_penalty %f -n -1 -p ",
		l.binPath,
		l.cpuThreadCount,
		l.gpuLayerCount,
		l.contextSize,
		options.TemperatureOrDefault(l.defaultTemperature),
		l.repeatPenalty,
	)
	l.logger.Log(fmt.Sprintf("llama.cpp command: \"%s\"", result))
	return result, nil
}

// We hook up to the llama.cpp binary by launching a subprocess and reading its standard output until
// processLineFunc(..) signals it should stop with false as the returned value.
// Launching it as a new subprocess for each run has the following benefits:
// - full isolation (for privacy)
// - fault-tolerance: crashes in llama.cpp (out of memory, segfaults, etc.) do not crash the AI agent altogether
func runInferCommand(cmdstr, prompt string, responseTimeout time.Duration, processLineFunc func(s string) bool) error {
	args := strings.Fields(cmdstr) // TODO probably unsafe, pass the arguments like we do it in llava.cpp
	args = append(args, prompt)
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(responseTimeout))
	defer cancelFunc()
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
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

func getAgentNameWithDelimiter(aiContext *domain.AIContext, promptFormatter domain.PromptFormatter) string {
	memories := []*domain.Memory{domain.NewMemory("", domain.MemoryTypeAction, aiContext.AgentName, time.Now(), "", "", nil)}
	result := strings.TrimSpace(promptFormatter.FormatDialog(memories))
	return result
}
