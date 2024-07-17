package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type CodeRunner struct {
	namedMutexAcquirer domain.NamedMutexAcquirer
}

func NewCodeRunner(
	namedMutexAcquirer domain.NamedMutexAcquirer,
) *CodeRunner {
	return &CodeRunner{
		namedMutexAcquirer: namedMutexAcquirer,
	}
}

func (c *CodeRunner) Run(code string) (string, error) {
	namedMutex, err := c.namedMutexAcquirer.AcquireNamedMutex("codePassDocker", time.Minute)
	if err != nil {
		return "", err
	}
	defer namedMutex.Release()
	err = c.preparePythonFile(code)
	if err != nil {
		return "", err
	}
	return c.runCodeInDocker()
}

func (c *CodeRunner) preparePythonFile(code string) error {
	file, err := os.Create("sandbox/code.py")
	if err != nil {
		return err
	}
	defer func() {
		_ = file.Close()
	}()
	_, err = file.WriteString(code)
	if err != nil {
		return err
	}
	return nil
}

func (c *CodeRunner) runCodeInDocker() (string, error) {
	cmd := exec.Command("docker", "run", "-v", fmt.Sprintf("%s/sandbox:/usr/src/app", os.Getenv("PWD")), "python:3-alpine", "python", "/usr/src/app/code.py") // create a pipe to capture the output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	err = cmd.Start()
	if err != nil {
		return "", err
	}
	output, err := io.ReadAll(stdout)
	if err != nil {
		return "", err
	}
	err = cmd.Wait()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
