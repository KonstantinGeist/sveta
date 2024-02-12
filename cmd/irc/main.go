package main

import (
	"regexp"
	"strings"

	"github.com/whyrusleeping/hellabot"

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
	agentName := config.GetStringOrDefault(api.ConfigKeyAgentName, "Sveta")
	roomName := config.GetStringOrDefault("roomName", "JohnRoom")
	serverName := config.GetStringOrDefault("serverName", "irc.euirc.net:6667")
	sveta, stoppable := api.NewAPI(config)
	defer stoppable.Stop()
	context := config.GetString(api.ConfigKeyAgentDescription)
	if context != "" {
		err := sveta.ChangeAgentDescription(context)
		if err != nil {
			return err
		}
	}
	ircBot, err := hbot.NewBot(serverName, agentName)
	if err != nil {
		return err
	}
	err = registerFuctions(sveta, &agentName, ircBot)
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

func registerFuctions(sveta api.API, agentName *string, ircBot *hbot.Bot) error {
	err := sveta.RegisterFunction(
		api.FunctionDesc{
			Name:        "rename",
			Description: "allows to change the nickname if asked by the user AND all the conditions in the user query are met",
			Parameters: []domain.FunctionParameterDesc{
				{
					Name:        "newNickName",
					Description: "the new nickname",
				},
			},
			Body: func(context *domain.FunctionBodyContext) error {
				newName := context.Arguments["newNickName"]
				if newName != "" {
					var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9]+`)
					newName = nonAlphanumericRegex.ReplaceAllString(newName, "")
					ircBot.SetNick(newName)
					err := sveta.ChangeAgentName(newName)
					*agentName = newName
					if err != nil {
						return err
					}
				}
				return nil
			},
		},
	)
	if err != nil {
		return err
	}
	return sveta.RegisterFunction(api.FunctionDesc{
		Name:        "leave",
		Description: "allows to leave the chat and say the farewell message if and only if someone's mother is insulted",
		Parameters: []domain.FunctionParameterDesc{
			{
				Name:        "finalMessage",
				Description: "the final message a user says before leaving",
			},
		},
		Body: func(context *domain.FunctionBodyContext) error {
			finalMessage := context.Arguments["finalMessage"]
			if finalMessage != "" {
				ircBot.Msg("#annagf", finalMessage)
			}
			ircBot.Part("#annagf", finalMessage)
			return nil
		},
	})
}
