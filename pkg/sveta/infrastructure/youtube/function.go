package youtube

import (
	"context"
	"fmt"
	"math/rand"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/api"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

func RegisterYoutubeSearchFunction(sveta api.API, config *common.Config) error {
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
		Body: func(input *api.FunctionInput) (api.FunctionOutput, error) {
			searchQuery := input.Arguments["searchQuery"]
			if searchQuery == "" {
				return api.FunctionOutput{}, nil
			}
			searchList := youtubeService.Search.List([]string{"id", "snippet"}).Q(searchQuery).MaxResults(20)
			searchListResponse, err := searchList.Do()
			if err != nil {
				return api.FunctionOutput{}, err
			}
			if len(searchListResponse.Items) == 0 {
				return api.FunctionOutput{}, nil
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
				return api.FunctionOutput{}, nil
			}
			item := items[rand.Intn(len(items))]
			return api.FunctionOutput{
				Output: fmt.Sprintf("According to Youtube search, here's a relevant video: https://youtube.com/watch?v=%s (title: \"%s\"). You MUST cite the video URL as is, so that the user could click on it.", item.Id.VideoId, item.Snippet.Title),
			}, nil
		},
	})
}
