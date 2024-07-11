package main

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/deepseekcoder"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/openmeteo"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/wolframalpha"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/youtube"
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
	err = registerFuctions(sveta, config)
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

func registerFuctions(sveta api.API, config *common.Config) error {
	err := openmeteo.RegisterWeatherFunction(sveta)
	if err != nil {
		return err
	}
	/*err = gocalculator.RegisterCalcFunction(sveta)
	if err != nil {
		return err
	}*/
	err = youtube.RegisterYoutubeSearchFunction(sveta, config)
	if err != nil {
		return err
	}
	err = deepseekcoder.RegisterCodeFunction(sveta)
	if err != nil {
		return err
	}
	return wolframalpha.RegisterWolframAlphaFunction(sveta, config)
}
