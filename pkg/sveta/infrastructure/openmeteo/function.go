package openmeteo

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

func RegisterWeatherFunction(sveta api.API) error {
	// OpenMeteo only works with latitudes/longitutes, so we have to maintain a separate list which matches
	// city names to their locations.
	latitudesAndLongitudesMap, err := readLatitudesAndLongitudes()
	if err != nil {
		return err
	}
	return sveta.RegisterFunction(api.FunctionDesc{
		Name:        "weather",
		Description: "allows to return information about weather if explicitly prompted by user",
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
				return domain.FunctionOutput{}, nil
			}
			latitudesAndLongitudes, ok := latitudesAndLongitudesMap[strings.ToLower(city)] // ToLower normalizes the city name
			if !ok {
				return domain.FunctionOutput{}, nil
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
		city := strings.ToLower(record[0]) // ToLower normalizes the city name
		_, ok := result[city]
		if ok {
			continue
		}
		result[city] = []string{record[1], record[2]}
	}
	return result, nil
}
