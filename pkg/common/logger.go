package common

import (
	"bufio"
	"fmt"
	"os"
)

type Logger interface {
	Log(message string)
}

type fileLogger struct {
	path       string
	fileWriter *bufio.Writer
}

// NewFileLogger logs to the file specified by `path`. If the file is unavailable, writes to the console.
func NewFileLogger(path string) Logger {
	return &fileLogger{
		path: path,
	}
}

func (f *fileLogger) Log(message string) {
	if f.fileWriterReady() {
		_, err := f.fileWriter.WriteString(message)
		if err != nil {
			f.logErrorToConsole(err.Error())
			f.logMessageToConsole(message)
		}
		err = f.fileWriter.Flush()
		if err != nil {
			f.logErrorToConsole(message)
		}
	} else {
		f.logMessageToConsole(message)
	}
}

func (f *fileLogger) logErrorToConsole(message string) {
	fmt.Printf("Error: %s. Logging switched to console.\n", message)
}

func (f *fileLogger) logMessageToConsole(message string) {
	fmt.Print(message)
}

func (f *fileLogger) fileWriterReady() bool {
	if f.fileWriter != nil {
		return true
	}
	file, err := os.OpenFile(f.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		f.logErrorToConsole(err.Error())
		return false
	}
	f.fileWriter = bufio.NewWriter(file)
	return true
}
