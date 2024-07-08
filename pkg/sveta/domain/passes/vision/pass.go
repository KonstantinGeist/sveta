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

type pass struct {
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

func NewPass(
	urlFinder common.URLFinder,
	visionModel Model,
	tempFilePathProvider common.TempFilePathProvider,
	config *common.Config,
	logger common.Logger,
) domain.Pass {
	return &pass{
		urlFinder:               urlFinder,
		visionModel:             visionModel,
		tempFilePathProvider:    tempFilePathProvider,
		whereToRememberedImages: make(map[string]*rememberedImageData),
		logger:                  logger,
		memoryDecayDuration:     config.GetIntOrDefault("imageMemoryDecayDuration", 3),
	}
}

func (p *pass) Capabilities() []*domain.Capability {
	return []*domain.Capability{
		{
			Name:        visionCapability,
			Description: "answers the user query by analyzing pictures (if provided)",
		},
	}
}

func (p *pass) Apply(context *domain.PassContext, nextPassFunc domain.NextPassFunc) error {
	if !context.IsCapabilityEnabled(visionCapability) {
		return nextPassFunc(context)
	}
	inputMemory := context.Memory(domain.DataKeyInput)
	if inputMemory == nil {
		return nextPassFunc(context)
	}
	var err error
	rememberedImage := p.getRememberedImage(inputMemory.Where)
	whatWithoutURL := inputMemory.What // first initialization, will be changed later
	urls := p.urlFinder.FindURLs(inputMemory.What)
	if len(urls) != 0 {
		url := urls[0] // let's do it with only one image so far
		if !common.IsImageFormat(url) {
			return nextPassFunc(context)
		}
		rememberedImage, err = p.rememberImage(inputMemory.Where, url)
		if err != nil {
			p.logger.Log(err.Error())
			inputMemory.What = fmt.Sprintf(couldntLoadImageFormatMessage, inputMemory.What)
			return nextPassFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
		}
		whatWithoutURL = p.removeURL(inputMemory.What, url)
	}
	if rememberedImage == nil || !rememberedImage.fileExists() {
		return nextPassFunc(context)
	}
	response, err := p.visionModel.Infer(rememberedImage.FilePath, inputMemory.What)
	if err != nil {
		p.logger.Log(err.Error())
		inputMemory.What = fmt.Sprintf(couldntLoadImageFormatMessage, inputMemory.What)
		return nextPassFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
	}
	inputMemory.What = fmt.Sprintf(imageDescriptionFormatMessage, rememberedImage.OriginalURL, response, whatWithoutURL)
	return nextPassFunc(context.WithMemory(domain.DataKeyInput, inputMemory))
}

func (p *pass) getRememberedImage(where string) *rememberedImageData {
	rememberedImage := p.whereToRememberedImages[where]
	if rememberedImage != nil {
		rememberedImage.MemoryDecayIndex--
		if rememberedImage.MemoryDecayIndex <= 0 {
			delete(p.whereToRememberedImages, where)
			rememberedImage = nil
		}
	}
	return rememberedImage
}

func (p *pass) rememberImage(where, url string) (*rememberedImageData, error) {
	result := &rememberedImageData{
		OriginalURL:      url,
		FilePath:         p.tempFilePathProvider.GetTempFilePath("image_" + common.Hash(where)),
		MemoryDecayIndex: p.memoryDecayDuration,
	}
	err := common.DownloadFromURL(url, result.FilePath)
	if err != nil {
		return nil, err
	}
	p.whereToRememberedImages[where] = result
	return result, nil
}

func (p *pass) removeURL(what, url string) string {
	return strings.TrimSpace(strings.ReplaceAll(what, url, ""))
}

func (r *rememberedImageData) fileExists() bool {
	_, err := os.Stat(r.FilePath)
	return err == nil
}
