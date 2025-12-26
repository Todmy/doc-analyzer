package visualization

import (
	"context"
	"fmt"
)

// SemanticAxis represents a user-defined semantic dimension
type SemanticAxis struct {
	Word      string    `json:"word"`
	Embedding []float32 `json:"-"` // Full embedding vector for projection
}

// PresetAxis represents a preset axis configuration
type PresetAxis struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Words       []string `json:"words"`
}

// DefaultPresets returns commonly used axis presets
func DefaultPresets() []PresetAxis {
	return []PresetAxis{
		{
			Name:        "abstract-concrete",
			Description: "Abstract concepts vs concrete implementations",
			Words:       []string{"abstract", "concrete"},
		},
		{
			Name:        "technical-simple",
			Description: "Technical complexity vs simplicity",
			Words:       []string{"technical", "simple"},
		},
		{
			Name:        "positive-negative",
			Description: "Positive vs negative sentiment",
			Words:       []string{"positive", "negative"},
		},
		{
			Name:        "theory-practice",
			Description: "Theoretical concepts vs practical application",
			Words:       []string{"theory", "practice"},
		},
	}
}

// EmbeddingProvider generates embeddings for words
type EmbeddingProvider interface {
	EmbedText(ctx context.Context, text string) ([]float32, error)
}

// SemanticProjector handles semantic axis projection
type SemanticProjector struct {
	embedder EmbeddingProvider
}

// NewSemanticProjector creates a new semantic projector
func NewSemanticProjector(embedder EmbeddingProvider) *SemanticProjector {
	return &SemanticProjector{embedder: embedder}
}

// FindSemanticAxis creates a semantic axis from a word
func (p *SemanticProjector) FindSemanticAxis(ctx context.Context, word string) (*SemanticAxis, error) {
	embedding, err := p.embedder.EmbedText(ctx, word)
	if err != nil {
		return nil, fmt.Errorf("embed word %q: %w", word, err)
	}

	return &SemanticAxis{
		Word:      word,
		Embedding: embedding,
	}, nil
}

// FindSemanticAxes creates multiple semantic axes from words
func (p *SemanticProjector) FindSemanticAxes(ctx context.Context, words []string) ([]SemanticAxis, error) {
	axes := make([]SemanticAxis, len(words))

	for i, word := range words {
		axis, err := p.FindSemanticAxis(ctx, word)
		if err != nil {
			return nil, err
		}
		axes[i] = *axis
	}

	return axes, nil
}

// ProjectToAxes projects embeddings onto semantic axes using dot product
func ProjectToAxes(embeddings [][]float32, axes []SemanticAxis) [][]float64 {
	if len(embeddings) == 0 || len(axes) == 0 {
		return nil
	}

	result := make([][]float64, len(embeddings))

	for i, emb := range embeddings {
		result[i] = make([]float64, len(axes))
		for j, axis := range axes {
			// Project using dot product (cosine similarity without normalization)
			result[i][j] = dotProduct(emb, axis.Embedding)
		}
	}

	// Normalize coordinates
	return normalizeCoordinates(result)
}

// dotProduct computes the dot product of two vectors
func dotProduct(a, b []float32) float64 {
	sum := float64(0)
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		sum += float64(a[i]) * float64(b[i])
	}
	return sum
}


// SemanticReducer implements Reducer using semantic axes
type SemanticReducer struct {
	axes []SemanticAxis
}

// NewSemanticReducer creates a reducer using semantic axes
func NewSemanticReducer(axes []SemanticAxis) *SemanticReducer {
	return &SemanticReducer{axes: axes}
}

// Name returns the reducer name
func (r *SemanticReducer) Name() string {
	return "semantic"
}

// Reduce projects embeddings onto semantic axes
func (r *SemanticReducer) Reduce(embeddings [][]float32, dims int) ([][]float64, error) {
	if len(r.axes) == 0 {
		return nil, fmt.Errorf("no semantic axes defined")
	}

	// Use only the requested number of dimensions
	axes := r.axes
	if dims < len(axes) {
		axes = axes[:dims]
	}

	return ProjectToAxes(embeddings, axes), nil
}
