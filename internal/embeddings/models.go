package embeddings

// Supported embedding models and their dimensions
const (
	ModelTextEmbedding3Small = "openai/text-embedding-3-small"
	ModelTextEmbedding3Large = "openai/text-embedding-3-large"
	ModelTextEmbeddingAda002 = "openai/text-embedding-ada-002"

	DimTextEmbedding3Small = 1536
	DimTextEmbedding3Large = 3072
	DimTextEmbeddingAda002 = 1536

	DefaultModel = ModelTextEmbedding3Small
)

// GetEmbeddingDimension returns the dimension for a given model
func GetEmbeddingDimension(model string) int {
	switch model {
	case ModelTextEmbedding3Small:
		return DimTextEmbedding3Small
	case ModelTextEmbedding3Large:
		return DimTextEmbedding3Large
	case ModelTextEmbeddingAda002:
		return DimTextEmbeddingAda002
	default:
		return DimTextEmbedding3Small
	}
}

// EmbeddingRequest represents a request to the embedding API
type EmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

// EmbeddingResponse represents the API response
type EmbeddingResponse struct {
	Data  []EmbeddingData `json:"data"`
	Model string          `json:"model"`
	Usage Usage           `json:"usage"`
}

// EmbeddingData represents a single embedding in the response
type EmbeddingData struct {
	Object    string    `json:"object"`
	Index     int       `json:"index"`
	Embedding []float32 `json:"embedding"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// Statement represents a text statement with its embedding
type Statement struct {
	ID        string
	Text      string
	Embedding []float32
}

// BatchResult represents the result of embedding a batch of texts
type BatchResult struct {
	Embeddings [][]float32
	TokensUsed int
	Errors     []error
}
