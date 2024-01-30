package aifilters

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/mvdan/xurls"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

// TODO

// TODO internationalize
const couldntLoadImageFormatMessage = "%s Description: \"no description because the URL failed to load\""
const imageDescriptionFormatMessage = "%s\nThe description of the picture says: \"%s\"\n%s"

type imageFilter struct {
	logger common.Logger
}

func NewImageFilter(logger common.Logger) domain.AIFilter {
	return &imageFilter{
		logger: logger,
	}
}

type saveOutput struct {
	savedOutput []byte
}

func (so *saveOutput) Write(p []byte) (n int, err error) {
	so.savedOutput = append(so.savedOutput, p...)
	return os.Stdout.Write(p)
}

func (i *imageFilter) Apply(who, what, where string, nextFilterFunc domain.NextFilterFunc) (string, error) {
	urls := xurls.Relaxed.FindAllString(what, -1)
	whatWithout := what
	url := "tmp.jpg"
	if len(urls) != 0 {
		url = urls[0] // let's do it with only one image so far
		if !strings.HasSuffix(url, ".jpg") && !strings.HasSuffix(url, ".jpeg") && !strings.HasSuffix(url, ".png") {
			return nextFilterFunc(who, what, where)
		}
		resp, err := http.Get(url)
		if err != nil {
			i.logger.Log(err.Error())
			return nextFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
		}
		defer func() {
			_ = resp.Body.Close()
		}()
		out, err := os.Create("tmp.jpg")
		if err != nil {
			i.logger.Log(err.Error())
			return nextFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			i.logger.Log(err.Error())
			return nextFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
		}
		whatWithout = strings.TrimSpace(strings.ReplaceAll(what, url, ""))
	}
	if _, err := os.Stat("tmp.jpg"); err != nil {
		return nextFilterFunc(who, what, where)
	}
	cmd := exec.Command("/home/konstantin/projects/sveta/bin/llava.cpp", "-m", "/home/konstantin/projects/sveta/bin/llava.bin", "--mmproj", "/home/konstantin/projects/sveta/bin/llava-proj.bin", "--image", "/home/konstantin/projects/sveta/bin/tmp.jpg", "--temp", "0.1", "-p", whatWithout)
	var so saveOutput
	cmd.Stdout = &so
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		i.logger.Log(err.Error())
		return nextFilterFunc(who, fmt.Sprintf(couldntLoadImageFormatMessage, what), where)
	}
	description := string(so.savedOutput)
	hackIndex := strings.Index(description, "per image patch)") // TODO
	if hackIndex != -1 {
		description = description[hackIndex+len("per image patch)"):]
	}
	description = strings.TrimSpace(description)
	return nextFilterFunc(who, fmt.Sprintf(imageDescriptionFormatMessage, url, description, whatWithout), where)
}
