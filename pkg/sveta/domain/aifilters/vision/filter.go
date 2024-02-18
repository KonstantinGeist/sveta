package vision

import (
	"fmt"
	"os"
	"strings"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const couldntLoadImageFormatMessage = "%s Description: \"no description because the URL failed to load\""
const imageDescriptionFormatMessage = "%s\nContext (description of the image): \"%s\"\nQuery: \"%s\" (if it's a question about the picture, use the provided context/description as is and nothing else, but slightly reformulate it in the language of your persona; otherwise, ignore the description)"

const visionCapability = "vision"

type filter struct {
	urlFinder               common.URLFinder
	visionModel             Model
	tempFilePathProvider    common.TempFilePathProvider
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
	tempFilePathProvider common.TempFilePathProvider,
	config *common.Config,
	logger common.Logger,
) domain.AIFilter {
	return &filter{
		urlFinder:               urlFinder,
		visionModel:             visionModel,
		tempFilePathProvider:    tempFilePathProvider,
		whereToRememberedImages: make(map[string]*rememberedImageData),
		logger:                  logger,
		memoryDecayDuration:     config.GetIntOrDefault("imageMemoryDecayDuration", 3),
	}
}

func (f *filter) Capabilities() []domain.AIFilterCapability {
	return []domain.AIFilterCapability{
		{
			Name:        visionCapability,
			Description: "answers the user query by analyzing pictures (if provided)",
		},
	}
}

func (f *filter) Apply(context *domain.AIFilterContext, nextAIFilterFunc domain.NextAIFilterFunc) error {
	if !context.IsCapabilityEnabled(visionCapability) {
		return nextAIFilterFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextAIFilterFunc(context)
	}
	var err error
	rememberedImage := f.getRememberedImage(inputMemory.Where)
	whatWithoutURL := inputMemory.What // first initialization, will be changed later
	urls := f.urlFinder.FindURLs(inputMemory.What)
	if len(urls) != 0 {
		url := urls[0] // let's do it with only one image so far
		if !common.IsImageFormat(url) {
			return nextAIFilterFunc(context)
		}
		rememberedImage, err = f.rememberImage(inputMemory.Where, url)
		if err != nil {
			f.logger.Log(err.Error())
			inputMemory.What = fmt.Sprintf(couldntLoadImageFormatMessage, inputMemory.What)
			return nextAIFilterFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
		}
		whatWithoutURL = f.removeURL(inputMemory.What, url)
	}
	if rememberedImage == nil || !rememberedImage.fileExists() {
		return nextAIFilterFunc(context)
	}
	response, err := f.visionModel.Infer(rememberedImage.FilePath, inputMemory.What)
	if err != nil {
		f.logger.Log(err.Error())
		inputMemory.What = fmt.Sprintf(couldntLoadImageFormatMessage, inputMemory.What)
		return nextAIFilterFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
	}
	inputMemory.What = fmt.Sprintf(imageDescriptionFormatMessage, rememberedImage.OriginalURL, response, whatWithoutURL)
	return nextAIFilterFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
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
		FilePath:         f.tempFilePathProvider.GetTempFilePath("image_" + common.Hash(where)),
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
