package image

type VisionModel interface {
	Infer(filePath, prompt string) (string, error)
}
