package wolframalpha

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func callShortAnswersAPI(query, apiKey string) (string, error) {
	queryParams := make(url.Values)
	queryParams.Add("appid", apiKey)
	queryParams.Add("i", query)
	apiURL := fmt.Sprintf("http://api.wolframalpha.com/v1/result?%s", queryParams.Encode())
	response, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = response.Body.Close()
	}()
	output, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	return string(output), nil
}
