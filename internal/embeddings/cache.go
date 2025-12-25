package embeddings

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
)

// Cache defines the interface for embedding cache
type Cache interface {
	// Get retrieves an embedding from cache
	Get(ctx context.Context, key string) ([]float32, bool, error)

	// Set stores an embedding in cache
	Set(ctx context.Context, key string, embedding []float32) error

	// GetMulti retrieves multiple embeddings from cache
	// Returns a map of key -> embedding for found entries
	GetMulti(ctx context.Context, keys []string) (map[string][]float32, error)

	// SetMulti stores multiple embeddings in cache
	SetMulti(ctx context.Context, embeddings map[string][]float32) error
}

// GenerateCacheKey creates a cache key from model and text
func GenerateCacheKey(model, text string) string {
	h := sha256.New()
	h.Write([]byte(model + ":" + text))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// CachedClient wraps a Client with caching
type CachedClient struct {
	client *Client
	cache  Cache
}

// NewCachedClient creates a new cached embedding client
func NewCachedClient(client *Client, cache Cache) *CachedClient {
	return &CachedClient{
		client: client,
		cache:  cache,
	}
}

// EmbedTexts generates embeddings with caching
func (c *CachedClient) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Generate cache keys
	keys := make([]string, len(texts))
	for i, text := range texts {
		keys[i] = GenerateCacheKey(c.client.model, text)
	}

	// Check cache
	cached, err := c.cache.GetMulti(ctx, keys)
	if err != nil {
		// Log error but continue without cache
		cached = make(map[string][]float32)
	}

	// Find uncached texts
	var uncachedTexts []string
	var uncachedIndices []int
	for i, key := range keys {
		if _, ok := cached[key]; !ok {
			uncachedTexts = append(uncachedTexts, texts[i])
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Generate embeddings for uncached texts
	var newEmbeddings [][]float32
	if len(uncachedTexts) > 0 {
		newEmbeddings, err = c.client.EmbedTexts(ctx, uncachedTexts)
		if err != nil {
			return nil, err
		}

		// Cache new embeddings
		toCache := make(map[string][]float32)
		for i, idx := range uncachedIndices {
			toCache[keys[idx]] = newEmbeddings[i]
		}
		if len(toCache) > 0 {
			_ = c.cache.SetMulti(ctx, toCache) // Ignore cache errors
		}
	}

	// Combine cached and new embeddings
	results := make([][]float32, len(texts))
	newIdx := 0
	for i, key := range keys {
		if emb, ok := cached[key]; ok {
			results[i] = emb
		} else {
			results[i] = newEmbeddings[newIdx]
			newIdx++
		}
	}

	return results, nil
}

// EmbedText generates an embedding for a single text with caching
func (c *CachedClient) EmbedText(ctx context.Context, text string) ([]float32, error) {
	results, err := c.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

// GetDimension returns the embedding dimension
func (c *CachedClient) GetDimension() int {
	return c.client.GetDimension()
}

// NoOpCache is a cache that doesn't cache anything (for testing)
type NoOpCache struct{}

func (c *NoOpCache) Get(ctx context.Context, key string) ([]float32, bool, error) {
	return nil, false, nil
}

func (c *NoOpCache) Set(ctx context.Context, key string, embedding []float32) error {
	return nil
}

func (c *NoOpCache) GetMulti(ctx context.Context, keys []string) (map[string][]float32, error) {
	return make(map[string][]float32), nil
}

func (c *NoOpCache) SetMulti(ctx context.Context, embeddings map[string][]float32) error {
	return nil
}
