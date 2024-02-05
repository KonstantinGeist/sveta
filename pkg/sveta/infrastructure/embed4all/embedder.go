package embed4all

import (
	"bytes"
	"os/exec"
	"strings"

	"kgeyst.com/sveta/pkg/sveta/domain"
)

type Embedder struct{}

// NewEmbedder depends on Python 3 and the Embed4all library.
// Also, it automatically downloads the sentence_transformers model which is not good for container-native images.
// TODO use something more robust and controllable, without a dependency on Python 3
func NewEmbedder() *Embedder {
	return &Embedder{}
}

func (v *Embedder) Embed(sentence string) (domain.Embedding, error) {
	if sentence == "" {
		return domain.Embedding{}, nil
	}
	cmd := exec.Command("python3", "embed.py", sentence)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return domain.Embedding{}, nil
	}
	result := out.String()
	const garbage = "bert_load_from_file: bert tokenizer vocab = 30522\n" // GPT4all outputs garbage so we want to strip it TODO
	index := strings.Index(result, garbage)
	if index != -1 {
		result = result[index+len(garbage):]
	}
	return domain.NewEmbeddingFromFormattedValues(result)
}
