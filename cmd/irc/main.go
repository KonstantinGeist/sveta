package main

import (
	"fmt"
	"strings"

	"github.com/whyrusleeping/hellabot"

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
	agentName := config.GetStringOrDefault(api.ConfigKeyAgentName, "Sveta")
	roomName := config.GetStringOrDefault("roomName", "JohnRoom")
	serverName := config.GetStringOrDefault("serverName", "irc.euirc.net:6667")
	lowerAgentName := strings.ToLower(agentName)
	sveta := api.NewAPI(config)
	context := config.GetString(api.ConfigKeyAgentDescription)
	if context != "" {
		err := sveta.ChangeAgentDescription(context)
		if err != nil {
			return err
		}
	}
	var trigger = hbot.Trigger{
		func(b *hbot.Bot, m *hbot.Message) bool {
			return true
		},
		func(b *hbot.Bot, m *hbot.Message) bool {
			if m.Command == "JOIN" && m.From != agentName {
				err := sveta.RememberAction(m.From, "I joined the chat", m.Content[1:])
				if err != nil {
					fmt.Println(err)
				}
				return true
			}
			if m.Command == "PART" && m.From != agentName {
				err := sveta.RememberAction(m.From, "I left the chat", m.Content[1:])
				if err != nil {
					fmt.Println(err)
				}
				return true
			}
			if m.Command != "PRIVMSG" {
				return true
			}
			if !strings.HasPrefix(strings.ToLower(m.Content), lowerAgentName) {
				return true
			}
			what := strings.TrimSpace(m.Content[len(agentName):])
			if len(what) == 0 || what[0] == '@' || len(m.To) == 0 || m.To[0] != '#' {
				return false
			}
			if what[0] == ',' {
				what = what[1:]
			}
			if what == "forget everything" {
				_ = sveta.ClearAllMemory()
				return false
			}
			if strings.HasPrefix(what, "context ") {
				context := what[len("context "):]
				_ = sveta.ChangeAgentDescription(context)
				return false
			}
			response, err := sveta.Respond(strings.TrimSpace(m.From), what, roomName)
			if err != nil {
				response = "I'm borked :("
			}
			b.Reply(m, m.From+" "+response)
			return true
		},
	}
	ircBot, err := hbot.NewBot(serverName, agentName)
	if err != nil {
		panic(err)
	}
	ircBot.AddTrigger(trigger)
	ircBot.Channels = []string{"#" + roomName}
	ircBot.Run()
	return nil
}
