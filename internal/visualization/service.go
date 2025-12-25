package visualization

import (
	"context"
	"fmt"
)

// Point represents a point in the visualization
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z,omitempty"`
}

// VisualizationResult holds the result of visualization
type VisualizationResult struct {
	Points     []Point        `json:"points"`
	Method     string         `json:"method"`
	Dimensions int            `json:"dimensions"`
	Axes       []SemanticAxis `json:"axes,omitempty"`
}

// Config holds visualization configuration
type Config struct {
	DefaultMethod     string
	DefaultDimensions int
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DefaultMethod:     "pca",
		DefaultDimensions: 2,
	}
}

// Service handles visualization generation
type Service struct {
	config    Config
	projector *SemanticProjector
}

// NewService creates a new visualization service
func NewService(config Config, embedder EmbeddingProvider) *Service {
	var projector *SemanticProjector
	if embedder != nil {
		projector = NewSemanticProjector(embedder)
	}

	return &Service{
		config:    config,
		projector: projector,
	}
}

// GetVisualization generates visualization coordinates for embeddings
func (s *Service) GetVisualization(
	ctx context.Context,
	embeddings [][]float32,
	method string,
	dimensions int,
	axisWords []string,
) (*VisualizationResult, error) {
	if len(embeddings) == 0 {
		return &VisualizationResult{
			Points:     []Point{},
			Method:     method,
			Dimensions: dimensions,
		}, nil
	}

	if method == "" {
		method = s.config.DefaultMethod
	}
	if dimensions == 0 {
		dimensions = s.config.DefaultDimensions
	}

	var reducer Reducer
	var axes []SemanticAxis

	switch method {
	case "pca":
		reducer = NewPCAReducer()
	case "semantic":
		if len(axisWords) == 0 {
			return nil, fmt.Errorf("semantic method requires axis words")
		}
		if s.projector == nil {
			return nil, fmt.Errorf("embedding provider not configured")
		}

		var err error
		axes, err = s.projector.FindSemanticAxes(ctx, axisWords)
		if err != nil {
			return nil, fmt.Errorf("find semantic axes: %w", err)
		}

		reducer = NewSemanticReducer(axes)
		dimensions = len(axes)
	default:
		return nil, fmt.Errorf("unknown method: %s", method)
	}

	coords, err := reducer.Reduce(embeddings, dimensions)
	if err != nil {
		return nil, fmt.Errorf("reduce: %w", err)
	}

	points := make([]Point, len(coords))
	for i, coord := range coords {
		p := Point{X: coord[0]}
		if len(coord) > 1 {
			p.Y = coord[1]
		}
		if len(coord) > 2 {
			p.Z = coord[2]
		}
		points[i] = p
	}

	return &VisualizationResult{
		Points:     points,
		Method:     method,
		Dimensions: dimensions,
		Axes:       axes,
	}, nil
}

// GetPresets returns available axis presets
func (s *Service) GetPresets() []PresetAxis {
	return DefaultPresets()
}
