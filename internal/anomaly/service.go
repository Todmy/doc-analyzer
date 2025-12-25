package anomaly

import (
	"github.com/todmy/doc-analyzer/pkg/models"
)

// DetectorType represents the type of anomaly detector to use
type DetectorType string

const (
	DetectorDistance  DetectorType = "distance"
	DetectorIsolation DetectorType = "isolation"
	DetectorEnsemble  DetectorType = "ensemble"
)

// Config holds anomaly detection service configuration
type Config struct {
	Detector   DetectorType
	K          int     // For distance-based (number of neighbors)
	NumTrees   int     // For isolation forest
	SampleSize int     // For isolation forest
	Threshold  float64 // Anomaly threshold (0-1)
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		Detector:   DetectorEnsemble,
		K:          5,
		NumTrees:   100,
		SampleSize: 256,
		Threshold:  0.7,
	}
}

// Service provides anomaly detection functionality
type Service struct {
	config             Config
	distanceDetector   *DistanceAnomalyDetector
	isolationDetector  *IsolationForest
}

// NewService creates a new anomaly detection service
func NewService(config Config) *Service {
	if config.K <= 0 {
		config.K = DefaultConfig().K
	}
	if config.NumTrees <= 0 {
		config.NumTrees = DefaultConfig().NumTrees
	}
	if config.SampleSize <= 0 {
		config.SampleSize = DefaultConfig().SampleSize
	}
	if config.Threshold <= 0 {
		config.Threshold = DefaultConfig().Threshold
	}

	return &Service{
		config:            config,
		distanceDetector:  NewDistanceAnomalyDetector(),
		isolationDetector: NewIsolationForest(config.NumTrees, config.SampleSize),
	}
}

// AnomalyResult represents an anomaly detection result
type AnomalyResult struct {
	Index      int
	Score      float64
	IsAnomaly  bool
	Text       string
	File       string
	Line       int
}

// DetectAnomalies detects anomalies in statements
func (s *Service) DetectAnomalies(statements []models.Statement) []AnomalyResult {
	if len(statements) == 0 {
		return []AnomalyResult{}
	}

	// Extract embeddings
	embeddings := make([][]float32, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding
	}

	// Get scores based on detector type
	var scores []float64
	switch s.config.Detector {
	case DetectorDistance:
		scores = s.distanceDetector.Detect(embeddings, s.config.K)
	case DetectorIsolation:
		s.isolationDetector.Fit(embeddings)
		scores = s.isolationDetector.Score(embeddings)
	case DetectorEnsemble:
		scores = s.ensembleScore(embeddings)
	default:
		scores = s.ensembleScore(embeddings)
	}

	// Build results
	results := make([]AnomalyResult, len(statements))
	for i, stmt := range statements {
		results[i] = AnomalyResult{
			Index:     i,
			Score:     scores[i],
			IsAnomaly: scores[i] >= s.config.Threshold,
			Text:      stmt.Text,
			File:      stmt.File,
			Line:      stmt.Line,
		}
	}

	return results
}

// GetAnomalies returns only statements flagged as anomalies
func (s *Service) GetAnomalies(statements []models.Statement) []AnomalyResult {
	allResults := s.DetectAnomalies(statements)

	var anomalies []AnomalyResult
	for _, r := range allResults {
		if r.IsAnomaly {
			anomalies = append(anomalies, r)
		}
	}

	return anomalies
}

// ensembleScore combines distance and isolation scores
func (s *Service) ensembleScore(embeddings [][]float32) []float64 {
	// Get distance-based scores
	distScores := s.distanceDetector.Detect(embeddings, s.config.K)

	// Get isolation forest scores
	s.isolationDetector.Fit(embeddings)
	isoScores := s.isolationDetector.Score(embeddings)

	// Combine with equal weights
	combined := make([]float64, len(embeddings))
	for i := range embeddings {
		combined[i] = (distScores[i] + isoScores[i]) / 2.0
	}

	return combined
}

// SetThreshold updates the anomaly threshold
func (s *Service) SetThreshold(threshold float64) {
	if threshold > 0 && threshold <= 1 {
		s.config.Threshold = threshold
	}
}

// GetThreshold returns the current anomaly threshold
func (s *Service) GetThreshold() float64 {
	return s.config.Threshold
}
