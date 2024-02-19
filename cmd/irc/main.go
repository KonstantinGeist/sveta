package main

import (
	"strings"

	"github.com/whyrusleeping/hellabot"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/gocalculator"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/openmeteo"
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
	agentName := config.GetStringOrDefault(api.ConfigKeyAgentName, "Sveta")
	roomName := config.GetStringOrDefault("roomName", "JohnRoom")
	serverName := config.GetStringOrDefault("serverName", "irc.euirc.net:6667")
	sveta, stoppable := api.NewAPI(config)
	defer stoppable.Stop()
	agentDescription := config.GetString(api.ConfigKeyAgentDescription)
	if agentDescription != "" {
		err := sveta.ChangeAgentDescription(agentDescription)
		if err != nil {
			return err
		}
	}
	ircBot, err := hbot.NewBot(serverName, agentName)
	if err != nil {
		return err
	}
	err = registerFuctions(sveta, config)
	if err != nil {
		return err
	}
	var trigger = hbot.Trigger{
		func(b *hbot.Bot, m *hbot.Message) bool {
			return true
		},
		func(b *hbot.Bot, m *hbot.Message) bool {
			if m.Command != "PRIVMSG" {
				return true
			}
			if !strings.HasPrefix(strings.ToLower(m.Content), strings.ToLower(agentName)) {
				return true
			}
			what := strings.TrimSpace(m.Content[len(agentName):])
			if len(what) == 0 || what[0] == '@' || len(m.To) == 0 || m.To[0] != '#' {
				return true
			}
			if what[0] == ',' {
				what = what[1:]
			}
			if what == "forget everything" {
				_ = sveta.ClearAllMemory()
				return true
			}
			if what == "summary" {
				summary, err := sveta.GetSummary(roomName)
				if err != nil || summary == "" {
					summary = "no summary"
				}
				b.Reply(m, m.From+" SUMMARY: "+summary)
				return true
			}
			if what == "list capabilities" {
				capabilities := strings.Join(sveta.ListCapabilities(), " ")
				b.Reply(m, m.From+" CAPABILITIES: "+capabilities)
				return true
			}
			if strings.HasPrefix(what, "context ") {
				context := what[len("context "):]
				_ = sveta.ChangeAgentDescription(context)
				return true
			}
			if strings.HasPrefix(what, "disable capability ") {
				capability := what[len("disable capability "):]
				err = sveta.EnableCapability(capability, false)
				if err == nil {
					b.Reply(m, m.From+" capability disabled")
				} else {
					b.Reply(m, m.From+" failed to disable capability")
				}
				return true
			}
			if strings.HasPrefix(what, "enable capability ") {
				capability := what[len("enable capability "):]
				err = sveta.EnableCapability(capability, true)
				if err == nil {
					b.Reply(m, m.From+" capability enabled")
				} else {
					b.Reply(m, m.From+" failed to enable capability")
				}
				return true
			}
			response, err := sveta.Respond(strings.TrimSpace(m.From), what, roomName)
			if err != nil {
				response = "I'm borked :("
			}
			if response != "" {
				b.Reply(m, m.From+" "+response)
			}
			return true
		},
	}
	ircBot.AddTrigger(trigger)
	ircBot.Channels = []string{"#" + roomName}
	ircBot.Run()
	return nil
}

func registerFuctions(sveta api.API, config *common.Config) error {
	err := openmeteo.RegisterWeatherFunction(sveta)
	if err != nil {
		return err
	}
	err = gocalculator.RegisterCalcFunction(sveta)
	if err != nil {
		return err
	}
	return youtube.RegisterYoutubeSearchFunction(sveta, config)
}
