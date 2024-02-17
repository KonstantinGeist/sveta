package domain

import (
	"fmt"
	"strings"
	"sync"
)

type FunctionDesc struct {
	Name        string
	Description string
	Parameters  []FunctionParameterDesc
	Body        FunctionBody
}

type FunctionBody func(context *FunctionInput) (FunctionOutput, error)

type FunctionParameterDesc struct {
	Name        string
	Description string
}

type FunctionInput struct {
	Arguments map[string]string
	Input     string
}

type FunctionOutput struct {
	Output string
	Stop   bool
}

type Closure struct {
	Name      string
	Arguments map[string]string
	Input     string
	Body      FunctionBody
}

type FunctionService struct {
	mutex           sync.Mutex
	aiContext       *AIContext
	responseService *ResponseService
	functionDescs   map[string]FunctionDesc
}

func NewFunctionService(
	aiContext *AIContext,
	responseService *ResponseService,
) *FunctionService {
	return &FunctionService{
		aiContext:       aiContext,
		responseService: responseService,
		functionDescs:   make(map[string]FunctionDesc),
	}
}

func (f *FunctionService) RegisterFunction(desc FunctionDesc) error {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.functionDescs[desc.Name] = desc
	return nil
}

func (f *FunctionService) CreateClosures(input string) ([]*Closure, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if len(f.functionDescs) == 0 { // short path
		return nil, nil
	}
	query := f.getQuery(input)
	type argumentOutput struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}
	type functionOutput struct {
		Name      string           `json:"name"`
		Arguments []argumentOutput `json:"arguments"`
	}
	var output struct {
		// To make answers more correct by forcing it to reason about the function name and
		// the arguments (Chain of Thought-like).
		Reasoning string           `json:"reasoningAboutFunctionAndArguments"`
		Functions []functionOutput `json:"functions"`
	}
	output.Functions = append(output.Functions, functionOutput{
		Name: "functionName",
	})
	output.Functions[0].Arguments = append(output.Functions[0].Arguments, argumentOutput{
		Name:  "argumentName",
		Value: "value",
	})
	err := f.getFunctionResponseService().RespondToQueryWithJSON(query, &output)
	if err != nil {
		return nil, err
	}
	closures := make([]*Closure, 0, len(output.Functions))
	for _, function := range output.Functions {
		functionDesc, ok := f.functionDescs[function.Name]
		if !ok {
			continue
		}
		argMap := make(map[string]string)
		for _, argument := range function.Arguments {
			if argument.Name != "argumentName" && argument.Value != "value" { // it may repeat the examples as is
				argMap[argument.Name] = argument.Value
			}
		}
		closures = append(closures, &Closure{
			Name:      function.Name,
			Arguments: argMap,
			Input:     input,
			Body:      functionDesc.Body,
		})
	}
	return closures, nil
}

func (c *Closure) Invoke() (FunctionOutput, error) {
	input := &FunctionInput{
		Arguments: c.Arguments,
		Input:     c.Input,
	}
	return c.Body(input)
}

func (f *FunctionService) getQuery(input string) string {
	var buf strings.Builder
	buf.WriteString("Below is a list of available Go functions: ```\n")
	for _, functionDesc := range f.functionDescs {
		buf.WriteString(fmt.Sprintf("// %s\n", functionDesc.Description))
		buf.WriteString(fmt.Sprintf("func %s(\n", functionDesc.Name))
		for _, parameterDesc := range functionDesc.Parameters {
			buf.WriteString(fmt.Sprintf("  %s string, // %s\n", parameterDesc.Name, parameterDesc.Description))
		}
		buf.WriteString(") {}\n\n")
	}
	buf.WriteString("```\n")
	buf.WriteString(fmt.Sprintf("List functions (and their arguments) which I need to call in order to satisfy the user query: \"%s\".\n", input))
	buf.WriteString("If no function satisfies the user query, return an empty list. Use only the JSON schema given above.")
	return buf.String()
}

func (f *FunctionService) getFunctionResponseService() *ResponseService {
	rankerAIContext := NewAIContext(
		f.aiContext.AgentName,
		"You're an intelligent assistant that tells which existing functions to call (if it's possible at all) based on the user query and a list of available Go functions. "+
			fmt.Sprintf("When outputting argument values, you take into consideration the following persona: \"%s\"", f.aiContext.AgentDescription),
		"",
	)
	return f.responseService.WithAIContext(rankerAIContext)
}
