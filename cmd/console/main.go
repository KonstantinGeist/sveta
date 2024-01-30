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
	sveta := api.NewAPI(config)
	context := config.GetString(api.ConfigKeyAgentDescription)
	if context != "" {
		err := sveta.ChangeAgentDescription(context)
		if err != nil {
			return err
		}
	}
	rl, err := readline.New("> ")
	if err != nil {
		return err
	}
	defer func() {
		_ = rl.Close()
	}()
	for {
		line, err := rl.Readline()
		if err != nil { // io.EOF
			break
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, ":") {
			line = line[1:]
			err := sveta.RememberAction(userName, line, roomName)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			response, err := sveta.Respond(userName, line, roomName)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Println(response)
		}
	}
	return nil
}
