package llavacpp

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var mutex sync.Mutex

type VisionModel struct{}

func NewVisionModel() *VisionModel {
	return &VisionModel{}
}

func (v *VisionModel) Infer(filePath, prompt string) (string, error) {
	// Only 1 request can be processed at a time currently because we run Sveta on commodity hardware which can't
	// usually process two requests simultaneously due to low amounts of VRAM.
	mutex.Lock()
	defer mutex.Unlock()
	cmd, err := buildExecCommand(filePath, prompt)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	result := out.String()
	return removeGarbage(result), nil
}

func buildExecCommand(filePath, what string) (*exec.Cmd, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return exec.Command(
		workingDirectory+"/llava.cpp",
		"-m", workingDirectory+"/llava.bin",
		"--mmproj", workingDirectory+"/llava-proj.bin",
		"--image", filePath,
		"--temp", "0.1",
		"-p", what,
	), nil
}

// TODO can we get rid of the hack?
func removeGarbage(result string) string {
	const anchor = "per image patch)"
	hackIndex := strings.Index(result, anchor)
	if hackIndex != -1 {
		result = result[hackIndex+len(anchor):]
	}
	return strings.TrimSpace(result)
}
