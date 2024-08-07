package main

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
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
	agentDescription := config.GetString(api.ConfigKeyAgentDescription)
	if agentDescription != "" {
		err := sveta.ChangeAgentDescription(agentDescription)
		if err != nil {
			return err
		}
	}
	var shouldStop bool
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
