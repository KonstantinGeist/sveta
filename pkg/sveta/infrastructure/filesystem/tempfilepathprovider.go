package filesystem

import (
	"fmt"
	"os"

	"kgeyst.com/sveta/pkg/common"
)

type TempFilePathProvider struct {
	tempDirectoryPath string
}

func NewTempFilePathProvider(config *common.Config) *TempFilePathProvider {
	return &TempFilePathProvider{
		tempDirectoryPath: config.GetStringOrDefault("tempFilePathProvider", os.TempDir()),
	}
}

func (t *TempFilePathProvider) GetTempFilePath(fileName string) string {
	return fmt.Sprintf("%s/%s", t.tempDirectoryPath, fileName)
}
