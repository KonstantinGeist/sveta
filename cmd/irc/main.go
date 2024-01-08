package main

import (
	"fmt"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"

	"github.com/whyrusleeping/hellabot"
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
	context := config.GetString(api.ConfigKeyContext)
	if context != "" {
		err := sveta.SetContext(context)
		if err != nil {
			return err
		}
	}
	err = sveta.LoadMemory("chunks.bin", "Context", roomName, time.Time{})
	if err != nil {
		fmt.Println(err)
	}
	var myTrigger = hbot.Trigger{
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
			if len(what) == 0 || what[0] == '@' {
				return false
			}
			if what[0] == ',' {
				what = what[1:]
			}
			if what == "forget everything" {
				_ = sveta.ForgetEverything()
				return false
			}
			if what == "summary" {
				summary, err := sveta.Summarize(roomName)
				if err != nil {
					fmt.Println(err)
				}
				b.Reply(m, m.From+" "+strings.TrimSpace(summary))
				return true
			}
			if strings.HasPrefix(what, "context ") {
				context := what[len("context "):]
				_ = sveta.SetContext(context)
				_ = sveta.ForgetEverything()
				return false
			}
			response, err := sveta.Reply(m.From, what, roomName)
			if err != nil {
				response = err.Error()
			}
			b.Reply(m, m.From+" "+strings.TrimSpace(response))
			return true
		},
	}
	ircBot, err := hbot.NewBot(serverName, agentName)
	if err != nil {
		panic(err)
	}
	ircBot.AddTrigger(myTrigger)
	ircBot.Channels = []string{"#" + roomName}
	ircBot.Run()
	return nil
}
