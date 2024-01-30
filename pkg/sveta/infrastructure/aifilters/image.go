package aifilters

import (
	"fmt"
	"os"
	"strings"

	"github.com/mvdan/xurls"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
	"kgeyst.com/sveta/pkg/sveta/infrastructure/llavacpp"
)

// TODO internationalize
const couldntLoadImageFormatMessage = "%s Description: \"no description because the URL failed to load\""
const imageDescriptionFormatMessage = "%s\nThe description of the picture says: \"%s\"\n%s"

const tempImagePath = "tmp.jpg"

type imageFilter struct {
	logger common.Logger
}

func NewImageFilter(logger common.Logger) domain.AIFilter {
	return &imageFilter{
		logger: logger,
	}
}

func (i *imageFilter) Apply(who, what, where string, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	urls := xurls.Relaxed.FindAllString(what, -1)
	whatWithoutURL := what
	url := tempImagePath
	if len(urls) != 0 {
		url = urls[0] // let's do it with only one image so far
		if !common.IsImageFormat(url) {
			return nextAIFilterFunc(who, what, where)
		}
		err := common.DownloadFromURL(url, tempImagePath)
		if err != nil {
			i.logger.Log(err.Error())
			return nextAIFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
		}
		whatWithoutURL = i.removeURL(what, url)
	}
	if _, err := os.Stat(tempImagePath); err != nil {
		return nextAIFilterFunc(who, what, where)
	}
	response, err := llavacpp.Run(tempImagePath, what)
	if err != nil {
		i.logger.Log(err.Error())
		return nextAIFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
	}
	return nextAIFilterFunc(who, fmt.Sprintf(imageDescriptionFormatMessage, url, response, whatWithoutURL), where)
}

func (i *imageFilter) removeURL(what, url string) string {
	return strings.TrimSpace(strings.ReplaceAll(what, url, ""))
}
