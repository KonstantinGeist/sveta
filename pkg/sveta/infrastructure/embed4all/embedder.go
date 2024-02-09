package embed4all

import (
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"kgeyst.com/sveta/pkg/common"
	"kgeyst.com/sveta/pkg/sveta/domain"
)

const beginTag = "<begin>"
const endTag = "<end>"
const embeddingDimensionCount = 384

type Embedder struct {
	mutex     sync.Mutex
	cmd       *exec.Cmd
	stdin     io.Writer
	stdout    io.Reader
	outBuffer []byte
	logger    common.Logger
}

// NewEmbedder depends on Python 3 and the Embed4all library.
// Also, it automatically downloads the sentence_transformers model which is not good for container-native images.
// TODO use something more robust and controllable, without a dependency on Python 3
func NewEmbedder(logger common.Logger) *Embedder {
	return &Embedder{
		outBuffer: make([]byte, 24000), // an estimate
		logger:    logger,
	}
}

func (v *Embedder) Embed(sentence string) (domain.Embedding, error) {
	sentence = strings.ReplaceAll(sentence, "\n", " ") // otherwise it can break line-by-line reading logic in embed.py
	v.logger.Log(fmt.Sprintf("Embedding: \"%s\"...\n", sentence))
	if sentence == "" {
		return domain.Embedding{}, nil
	}
	v.mutex.Lock()
	defer v.mutex.Unlock()
	err := v.startSubprocessIfRequired()
	if err != nil {
		v.logger.Log("failed to embed: " + err.Error())
		return domain.Embedding{}, nil
	}
	_, err = v.stdin.Write([]byte(fmt.Sprintf("%s\n\n", sentence)))
	if err != nil {
		v.logger.Log("failed to embed (writing to embed4all): " + err.Error())
		return domain.Embedding{}, nil
	}
	var result string
	for {
		n, err := v.stdout.Read(v.outBuffer)
		if err != nil {
			v.logger.Log("failed to embed (reading from embed4all): " + err.Error())
			return domain.Embedding{}, nil
		}
		if n == 0 {
			time.Sleep(time.Millisecond)
			continue
		}
		line := string(v.outBuffer[:n])
		result += line
		if strings.Contains(line, "<end>") {
			break
		}
	}
	beginIndex := strings.Index(result, beginTag)
	endIndex := strings.Index(result, endTag)
	if beginIndex == -1 || endIndex == -1 || beginIndex >= endIndex {
		v.logger.Log("failed to embed (missing begin/end)")
		return domain.Embedding{}, nil
	}
	result = result[beginIndex+len(beginTag) : endIndex]
	embedding, err := domain.NewEmbeddingFromFormattedValues(result)
	if embedding.DimensionCount() != embeddingDimensionCount {
		v.logger.Log("wrong dimension count")
		return domain.Embedding{}, nil
	}
	return embedding, err
}

func (v *Embedder) startSubprocessIfRequired() error {
	if v.cmd != nil {
		return nil
	}
	cmd := exec.Command("python3", "embed.py")
	stdin, _ := cmd.StdinPipe()
	stdout, _ := cmd.StdoutPipe()
	err := cmd.Start()
	if err != nil {
		return err
	}
	v.cmd = cmd
	v.stdin = stdin
	v.stdout = stdout
	return nil
}
