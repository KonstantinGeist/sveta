package common

type TempFilePathProvider interface {
	GetTempFilePath(fileName string) string
}
