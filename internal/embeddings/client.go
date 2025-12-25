package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	defaultBaseURL       = "https://openrouter.ai/api/v1"
	defaultBatchSize     = 100
	defaultMaxConcurrent = 5
	defaultTimeout       = 30 * time.Second
)

// Client handles embedding generation via OpenRouter API
type Client struct {
	httpClient    *http.Client
	baseURL       string
	apiKey        string
	model         string
	batchSize     int
	maxConcurrent int
}

// ClientOption configures the Client
type ClientOption func(*Client)

// WithBaseURL sets a custom base URL
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithModel sets the embedding model
func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

// WithBatchSize sets the batch size for API requests
func WithBatchSize(size int) ClientOption {
	return func(c *Client) {
		c.batchSize = size
	}
}

// WithMaxConcurrent sets the max concurrent requests
func WithMaxConcurrent(n int) ClientOption {
	return func(c *Client) {
		c.maxConcurrent = n
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = d
	}
}

// NewClient creates a new embedding client
func NewClient(apiKey string, opts ...ClientOption) *Client {
	c := &Client{
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		baseURL:       defaultBaseURL,
		apiKey:        apiKey,
		model:         DefaultModel,
		batchSize:     defaultBatchSize,
		maxConcurrent: defaultMaxConcurrent,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// EmbedTexts generates embeddings for a list of texts
func (c *Client) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	// Split into batches
	batches := c.splitIntoBatches(texts)
	results := make([][]float32, len(texts))

	// Process batches with concurrency control
	sem := make(chan struct{}, c.maxConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	resultOffset := 0
	for batchIdx, batch := range batches {
		wg.Add(1)
		batchStart := resultOffset
		resultOffset += len(batch)

		go func(idx int, batch []string, start int) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire
			defer func() { <-sem }() // Release

			embeddings, err := c.embedBatch(ctx, batch)

			mu.Lock()
			defer mu.Unlock()

			if err != nil && firstErr == nil {
				firstErr = fmt.Errorf("batch %d: %w", idx, err)
				return
			}

			for i, emb := range embeddings {
				results[start+i] = emb
			}
		}(batchIdx, batch, batchStart)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

// EmbedText generates an embedding for a single text
func (c *Client) EmbedText(ctx context.Context, text string) ([]float32, error) {
	results, err := c.EmbedTexts(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return results[0], nil
}

// GetDimension returns the embedding dimension for the configured model
func (c *Client) GetDimension() int {
	return GetEmbeddingDimension(c.model)
}

func (c *Client) splitIntoBatches(texts []string) [][]string {
	var batches [][]string
	for i := 0; i < len(texts); i += c.batchSize {
		end := i + c.batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batches = append(batches, texts[i:end])
	}
	return batches
}

func (c *Client) embedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := EmbeddingRequest{
		Model: c.model,
		Input: texts,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/embeddings", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(body, &embResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Sort by index to ensure order matches input
	embeddings := make([][]float32, len(texts))
	for _, data := range embResp.Data {
		if data.Index < len(embeddings) {
			embeddings[data.Index] = data.Embedding
		}
	}

	return embeddings, nil
}
