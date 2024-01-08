package domain

type Embedder interface {
	// Embed calculates an embedding (a coordinate in a virtual semantic space) of a sentence (not only individual words).
	// The produced embeddings can be compared with Embedding.GetSimilarityTo(..)
	Embed(sentence string) (Embedding, error)
}
