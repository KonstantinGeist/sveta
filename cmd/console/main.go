package main

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

func main() {
	err := mainImpl()
	if err != nil {
		panic(err)
	}
}

func mainImpl() error {
	config, err := common.LoadConfig("config.yaml")
	if err != nil {
		return err
	}
	userName := config.GetStringOrDefault("userName", "John")
	roomName := config.GetStringOrDefault("roomName", "JohnRoom")
	sveta, stoppable := api.NewAPI(config)
	defer stoppable.Stop()
	context := config.GetString(api.ConfigKeyAgentDescription)
	if context != "" {
		err := sveta.ChangeAgentDescription(context)
		if err != nil {
			return err
		}
	}
	var shouldStop bool
	err = registerFuctions(sveta, &shouldStop)
	if err != nil {
		return err
	}
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	defer func() {
		_ = rl.Close()
	}()
	for !shouldStop {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		line = strings.TrimSpace(line)
		response, err := sveta.Respond(userName, line, roomName)
		if err != nil {
			response = "I'm borked :("
		}
		if response != "" {
			fmt.Println(response)
		}
	}
	return nil
}

func registerFuctions(sveta api.API, shouldStop *bool) error {
	return sveta.RegisterFunction(api.FunctionDesc{
		Name:        "leave",
		Description: "allows to leave the chat and say the farewell message if insulted",
		Parameters: []domain.FunctionParameterDesc{
			{
				Name:        "finalMessage",
				Description: "the final message a user says before leaving",
			},
		},
		Body: func(context *domain.FunctionBodyContext) error {
			finalMessage := context.Arguments["finalMessage"]
			if finalMessage != "" {
				fmt.Println(finalMessage)
			}
			*shouldStop = true
			return nil
		},
	})
}
