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
const imageDescriptionFormatMessage = "%s\nThe description of the picture says: \"%s\"\n%s (when answering, use only the description above and nothing else, but use language which is appropriate for your persona)"

type imageFilter struct {
	whereToRememberedImages map[string]*rememberedImageData
	logger                  common.Logger
	memoryDecayDuration     int
}

type rememberedImageData struct {
	OriginalURL      string
	FilePath         string
	MemoryDecayIndex int
}

func NewImageFilter(config *common.Config, logger common.Logger) domain.AIFilter {
	return &imageFilter{
		whereToRememberedImages: make(map[string]*rememberedImageData),
		logger:                  logger,
		memoryDecayDuration:     config.GetIntOrDefault("imageMemoryDecayDuration", 3),
	}
}

func (i *imageFilter) Apply(who, what, where string, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	var err error
	rememberedImage := i.getRememberedImage(where)
	whatWithoutURL := what // first initialization, will be changed later
	urls := xurls.Relaxed.FindAllString(what, -1)
	if len(urls) != 0 {
		url := urls[0] // let's do it with only one image so far
		if !common.IsImageFormat(url) {
			return nextAIFilterFunc(who, what, where)
		}
		rememberedImage, err = i.rememberImage(where, url)
		if err != nil {
			i.logger.Log(err.Error())
			return nextAIFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
		}
		whatWithoutURL = i.removeURL(what, url)
	}
	if rememberedImage == nil || !rememberedImage.fileExists() {
		return nextAIFilterFunc(who, what, where)
	}
	response, err := llavacpp.Run(rememberedImage.FilePath, what)
	if err != nil {
		i.logger.Log(err.Error())
		return nextAIFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
	}
	return nextAIFilterFunc(who, fmt.Sprintf(imageDescriptionFormatMessage, rememberedImage.OriginalURL, response, whatWithoutURL), where)
}

func (i *imageFilter) getRememberedImage(where string) *rememberedImageData {
	rememberedImage := i.whereToRememberedImages[where]
	if rememberedImage != nil {
		rememberedImage.MemoryDecayIndex--
		if rememberedImage.MemoryDecayIndex <= 0 {
			delete(i.whereToRememberedImages, where)
			rememberedImage = nil
		}
	}
	return rememberedImage
}

func (i *imageFilter) rememberImage(where, url string) (*rememberedImageData, error) {
	result := &rememberedImageData{
		OriginalURL:      url,
		FilePath:         os.TempDir() + "/svpc_" + common.Hash(where),
		MemoryDecayIndex: i.memoryDecayDuration,
	}
	err := common.DownloadFromURL(url, result.FilePath)
	if err != nil {
		return nil, err
	}
	i.whereToRememberedImages[where] = result
	return result, nil
}

func (i *imageFilter) removeURL(what, url string) string {
	return strings.TrimSpace(strings.ReplaceAll(what, url, ""))
}

func (r *rememberedImageData) fileExists() bool {
	_, err := os.Stat(r.FilePath)
	return err == nil
}
