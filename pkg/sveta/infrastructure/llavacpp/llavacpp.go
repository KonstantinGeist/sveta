package llavacpp

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
)

func Run(filePath, what string) (string, error) {
	cmd, err := buildExecCommand(filePath, what)
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
	hackIndex := strings.Index(result, "per image patch)") // TODO
	if hackIndex != -1 {
		result = result[hackIndex+len("per image patch)"):]
	}
	return strings.TrimSpace(result), nil
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
