package embed4all

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

type Embedder struct {
	logger          common.Logger
	responseTimeout time.Duration
}

// NewEmbedder depends on Python 3 and the Embed4all library.
// Also, it automatically downloads the sentence_transformers model which is not good for container-native images.
// TODO use something more robust and controllable, without a dependency on Python 3
func NewEmbedder(
	config *common.Config,
	logger common.Logger,
) *Embedder {
	return &Embedder{
		logger:          logger,
		responseTimeout: config.GetDurationOrDefault("embedResponseTimeout", time.Second*20),
	}
}

func (v *Embedder) Embed(sentence string) (domain.Embedding, error) {
	v.logger.Log(fmt.Sprintf("Embedding: \"%s\"", sentence))
	if sentence == "" {
		return domain.Embedding{}, nil
	}
	var out bytes.Buffer
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(v.responseTimeout))
	defer cancelFunc()
	cmd := exec.CommandContext(ctx, "python3", "embed.py", sentence)
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		v.logger.Log(err.Error())
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
