package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mnogu/go-calculator"
	"github.com/whyrusleeping/hellabot"
	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

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
	err = registerFuctions(sveta, &agentName, config, ircBot)
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

func registerFuctions(sveta api.API, agentName *string, config *common.Config, ircBot *hbot.Bot) error {
	latitudesAndLongitudesMap, err := readLatitudesAndLongitudes()
	if err != nil {
		return err
	}
	err = sveta.RegisterFunction(api.FunctionDesc{
		Name:        "weather",
		Description: "allows to return information about weather if prompted by user",
		Parameters: []domain.FunctionParameterDesc{
			{
				Name:        "city",
				Description: "the name of the city for which weather information is required",
			},
		},
		Body: func(context *api.FunctionInput) (api.FunctionOutput, error) {
			city := context.Arguments["city"]
			city = common.RemoveDoubleQuotesIfAny(city)
			city = common.RemoveSingleQuotesIfAny(city)
			spaceIndex := strings.Index(city, " ")
			if spaceIndex != -1 { // for stuff like "Washington, DC"
				city = city[0:spaceIndex]
				city = strings.ReplaceAll(city, ",", " ")
			}
			if city == "" {
				return domain.FunctionOutput{NoOutput: true}, nil // current=temperature_1ly=temperature_1m,relative_humidity_1m,wind_speed_1m"y_1m,wind_speed_1m
			}
			latitudesAndLongitudes, ok := latitudesAndLongitudesMap[strings.ToLower(city)]
			if !ok {
				return domain.FunctionOutput{NoOutput: true}, nil
			}
			rawData, err := common.ReadAllFromURL(fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%s&longitude=%s&current=temperature_2m", latitudesAndLongitudes[0], latitudesAndLongitudes[1]))
			if err != nil {
				return domain.FunctionOutput{}, err
			}
			var output struct {
				Current struct {
					Temperature float64 `json:"temperature_2m"`
				} `json:"current"`
			}
			err = json.Unmarshal(rawData, &output)
			if err != nil {
				return domain.FunctionOutput{}, err
			}
			return domain.FunctionOutput{
				Output: fmt.Sprintf("Temperature in %s is currently %sC", city, strconv.FormatFloat(output.Current.Temperature, 'f', -1, 64)),
			}, nil
		},
	})
	if err != nil {
		return err
	}
	err = sveta.RegisterFunction(api.FunctionDesc{
		Name:        "calc",
		Description: "allows to calculate a math expression if the user query requires it",
		Parameters: []domain.FunctionParameterDesc{
			{
				Name:        "mathExpression",
				Description: "math expression, such as (2 + 3 * 4)",
			},
		},
		Body: func(context *api.FunctionInput) (api.FunctionOutput, error) {
			mathExpression := context.Arguments["mathExpression"]
			mathExpression = common.RemoveSingleQuotesIfAny(mathExpression)
			mathExpression = common.RemoveDoubleQuotesIfAny(mathExpression)
			if mathExpression == "" {
				return domain.FunctionOutput{NoOutput: true}, nil
			}
			value, err := calculator.Calculate(mathExpression)
			if err != nil {
				return domain.FunctionOutput{NoOutput: true}, nil
			}
			return domain.FunctionOutput{
				Output: fmt.Sprintf("According to the calculator, the result of the user query below is %s (calculated based on the math formula \"%s\", which is needed to answer the user query). This result takes precedence over any other possible result. Never calculate the result yourself, just cite the result as is, because it's the only correct option. Never mention the calculator, pretend as if you came to this result all by yourself.", strconv.FormatFloat(value, 'f', -1, 64), mathExpression),
			}, nil
		},
	})
	if err != nil {
		return err
	}
	youtubeAPIKey := config.GetString("youtubeAPIKey")
	if youtubeAPIKey == "" {
		return nil
	}
	youtubeService, err := youtube.NewService(context.Background(), option.WithAPIKey(youtubeAPIKey))
	if err != nil {
		return err
	}
	return sveta.RegisterFunction(api.FunctionDesc{
		Name:        "youtube",
		Description: "allows to search for a video on Youtube if the user is explicitly looking for a video or music",
		Parameters: []domain.FunctionParameterDesc{
			{
				Name:        "searchQuery",
				Description: "the query to use",
			},
		},
		Body: func(context *api.FunctionInput) (api.FunctionOutput, error) {
			searchQuery := context.Arguments["searchQuery"]
			if searchQuery == "" {
				return api.FunctionOutput{NoOutput: true}, nil
			}
			searchList := youtubeService.Search.List([]string{"id", "snippet"}).Q(searchQuery).MaxResults(20)
			searchListResponse, err := searchList.Do()
			if err != nil {
				return api.FunctionOutput{NoOutput: true}, nil
			}
			if len(searchListResponse.Items) == 0 {
				return api.FunctionOutput{NoOutput: true}, nil
			}
			var items []*youtube.SearchResult
			for _, item := range searchListResponse.Items {
				if item.Id.VideoId != "" {
					items = append(items, item)
					if len(items) == 3 {
						break
					}
				}
			}
			if len(items) == 0 {
				return api.FunctionOutput{NoOutput: true}, nil
			}
			item := items[rand.Intn(len(items))]
			return api.FunctionOutput{
				Output: fmt.Sprintf("According to Youtube search, here's a relevant video: https://youtube.com/watch?v=%s (title: \"%s\"). You MUST cite the video URL as is, so that the user could click on it.", item.Id.VideoId, item.Snippet.Title),
			}, nil
		},
	})
}

func readLatitudesAndLongitudes() (map[string][]string, error) {
	path, err := filepath.Abs("../data/worldcities.csv")
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = f.Close()
	}()
	result := make(map[string][]string)
	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		city := strings.ToLower(record[0])
		_, ok := result[city]
		if ok {
			continue
		}
		result[city] = []string{record[1], record[2]}
	}
	return result, nil
}
