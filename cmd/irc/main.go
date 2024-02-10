package main

import (
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
	sveta, stoppable := api.NewAPI(config)
	defer stoppable.Stop()
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
			if what == "summary" {
				summary, err := sveta.GetSummary(roomName)
				if err != nil || summary == "" {
					summary = "no summary"
				}
				b.Reply(m, m.From+" SUMMARY: "+summary)
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
