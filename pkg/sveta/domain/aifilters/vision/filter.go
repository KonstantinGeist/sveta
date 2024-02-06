package vision

import (
	"fmt"
	"os"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO internationalize
const couldntLoadImageFormatMessage = "%s Description: \"no description because the URL failed to load\""
const imageDescriptionFormatMessage = "%s\nContext (description of the image): \"%s\"\nQuery: \"%s\" (if it's a question about the picture, use the provided context/description as is and nothing else, but slightly reformulate it in the language of your persona; otherwise, ignore the description)"

type filter struct {
	urlFinder               common.URLFinder
	visionModel             Model
	whereToRememberedImages map[string]*rememberedImageData
	logger                  common.Logger
	memoryDecayDuration     int
}

type rememberedImageData struct {
	OriginalURL      string
	FilePath         string
	MemoryDecayIndex int
}

func NewFilter(
	urlFinder common.URLFinder,
	visionModel Model,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		urlFinder:               urlFinder,
		visionModel:             visionModel,
		whereToRememberedImages: make(map[string]*rememberedImageData),
		logger:                  logger,
		memoryDecayDuration:     config.GetIntOrDefault("imageMemoryDecayDuration", 3),
	}
}

func (f *filter) Apply(context domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) (string, error) {
	var err error
	rememberedImage := f.getRememberedImage(context.Where)
	whatWithoutURL := context.What // first initialization, will be changed later
	urls := f.urlFinder.FindURLs(context.What)
	if len(urls) != 0 {
		url := urls[0] // let's do it with only one image so far
		if !common.IsImageFormat(url) {
			return nextAIFilterFunc(context)
		}
		rememberedImage, err = f.rememberImage(context.Where, url)
		if err != nil {
			f.logger.Log(err.Error())
			return nextAIFilterFunc(context.WithWhat(fmt.Sprintf(couldntLoadImageFormatMessage, context.What)))
		}
		whatWithoutURL = f.removeURL(context.What, url)
	}
	if rememberedImage == nil || !rememberedImage.fileExists() {
		return nextAIFilterFunc(context)
	}
	response, err := f.visionModel.Infer(rememberedImage.FilePath, context.What)
	if err != nil {
		f.logger.Log(err.Error())
		return nextAIFilterFunc(context.WithWhat(fmt.Sprintf(couldntLoadImageFormatMessage, context.What)))
	}
	return nextAIFilterFunc(context.WithWhat(fmt.Sprintf(imageDescriptionFormatMessage, rememberedImage.OriginalURL, response, whatWithoutURL)))
}

func (f *filter) getRememberedImage(where string) *rememberedImageData {
	rememberedImage := f.whereToRememberedImages[where]
	if rememberedImage != nil {
		rememberedImage.MemoryDecayIndex--
		if rememberedImage.MemoryDecayIndex <= 0 {
			delete(f.whereToRememberedImages, where)
			rememberedImage = nil
		}
	}
	return rememberedImage
}

func (f *filter) rememberImage(where, url string) (*rememberedImageData, error) {
	result := &rememberedImageData{
		OriginalURL:      url,
		FilePath:         os.TempDir() + "/svpc_" + common.Hash(where),
		MemoryDecayIndex: f.memoryDecayDuration,
	}
	err := common.DownloadFromURL(url, result.FilePath)
	if err != nil {
		return nil, err
	}
	f.whereToRememberedImages[where] = result
	return result, nil
}

func (f *filter) removeURL(what, url string) string {
	return strings.TrimSpace(strings.ReplaceAll(what, url, ""))
}

func (r *rememberedImageData) fileExists() bool {
	_, err := os.Stat(r.FilePath)
	return err == nil
}
