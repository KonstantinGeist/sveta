package vision

type Model interface {
	Infer(filePath, prompt string) (string, error)
}
