package domain

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Embedding a coordinate in a virtual embedding space. Embeddings can be compared to find which sentences are close
// to each other in meaning.
type Embedding struct {
	values []float64
}

// NewEmbedding creates a new embedding from the provided vector components (can be an arbitrary number, depends on the
// exact embedding model used).
func NewEmbedding(values []float64) Embedding {
	return Embedding{values: values}
}

// NewEmbeddingFromFormattedValues creates an embedding from a text form.
// Example format: "0.123 0.345 0.678" etc. where each float value corresponds to a vector component.
func NewEmbeddingFromFormattedValues(text string) (Embedding, error) {
	str := strings.TrimSpace(text)
	split := strings.Split(str, " ")
	values := make([]float64, len(split))
	for i := 0; i < len(split); i++ {
		value, err := strconv.ParseFloat(split[i], 64)
		if err != nil {
			return Embedding{}, err
		}
		values[i] = value
	}
	return NewEmbedding(values), nil
}

func (a Embedding) ToFormattedValues() string {
	var builder strings.Builder
	for i := 0; i < len(a.values); i++ {
		builder.WriteString(fmt.Sprintf("%.3f", a.values[i]))
		if i < len(a.values)-1 {
			builder.WriteRune(' ')
		}
	}
	return builder.String()
}

// GetSimilarityTo finds how similar two embeddings are to each other semantically using cosine similarity.
// If the result value is 1.0 -- the embeddings are identical. If it's 0.0 -- the embeddings are completely different.
func (a Embedding) GetSimilarityTo(b Embedding) float64 {
	aValues := a.values
	bValues := b.values
	if len(aValues) != len(bValues) {
		return 0.0
	}
	var sum, s1, s2 float64
	for i := 0; i < len(aValues); i++ {
		sum += aValues[i] * bValues[i]
		s1 += math.Pow(aValues[i], 2)
		s2 += math.Pow(bValues[i], 2)
	}
	if s1 == 0 || s2 == 0 {
		return 0.0
	}
	return sum / (math.Sqrt(s1) * math.Sqrt(s2))
}

func (a Embedding) GetBestSimilarityTo(bs []Embedding) float64 {
	var bestSimilarity = 0.0
	for _, b := range bs {
		similarity := a.GetSimilarityTo(b)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
		}
	}
	return bestSimilarity
}

func (a Embedding) DimensionCount() int {
	return len(a.values)
}
