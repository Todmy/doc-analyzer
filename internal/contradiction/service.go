package contradiction

import (
	"context"
	"sort"
)

// Service provides high-level contradiction detection
type Service struct {
	analyzer *Analyzer
	config   ServiceConfig
}

// ServiceConfig holds service configuration
type ServiceConfig struct {
	MaxPairsToAnalyze int
	MinSimilarity     float64
	MaxConcurrent     int
}

// DefaultServiceConfig returns default service configuration
func DefaultServiceConfig() ServiceConfig {
	return ServiceConfig{
		MaxPairsToAnalyze: 100,
		MinSimilarity:     0.5,
		MaxConcurrent:     5,
	}
}

// NewService creates a new contradiction detection service
func NewService(analyzer *Analyzer, config ServiceConfig) *Service {
	if config.MaxPairsToAnalyze <= 0 {
		config.MaxPairsToAnalyze = DefaultServiceConfig().MaxPairsToAnalyze
	}
	if config.MinSimilarity <= 0 {
		config.MinSimilarity = DefaultServiceConfig().MinSimilarity
	}
	if config.MaxConcurrent <= 0 {
		config.MaxConcurrent = DefaultServiceConfig().MaxConcurrent
	}

	return &Service{
		analyzer: analyzer,
		config:   config,
	}
}

// DetectContradictions finds contradictions in statement pairs
func (s *Service) DetectContradictions(ctx context.Context, pairs []StatementPair) ([]ContradictionResult, error) {
	// Filter pairs by similarity threshold
	filtered := filterPairs(pairs, s.config.MinSimilarity)

	// Limit number of pairs to analyze
	if len(filtered) > s.config.MaxPairsToAnalyze {
		// Sort by similarity and take top N
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].Similarity > filtered[j].Similarity
		})
		filtered = filtered[:s.config.MaxPairsToAnalyze]
	}

	// Analyze pairs
	results, err := s.analyzer.AnalyzePairs(ctx, filtered, s.config.MaxConcurrent)
	if err != nil {
		return nil, err
	}

	// Sort results by severity
	sort.Slice(results, func(i, j int) bool {
		return severityOrder(results[i].Severity) > severityOrder(results[j].Severity)
	})

	return results, nil
}

// GroupBySeverity groups contradictions by severity level
func GroupBySeverity(results []ContradictionResult) map[Severity][]ContradictionResult {
	grouped := make(map[Severity][]ContradictionResult)

	for _, r := range results {
		grouped[r.Severity] = append(grouped[r.Severity], r)
	}

	return grouped
}

// GroupByType groups contradictions by type
func GroupByType(results []ContradictionResult) map[ContradictionType][]ContradictionResult {
	grouped := make(map[ContradictionType][]ContradictionResult)

	for _, r := range results {
		grouped[r.Type] = append(grouped[r.Type], r)
	}

	return grouped
}

// GroupByFile groups contradictions by file
func GroupByFile(results []ContradictionResult) map[string][]ContradictionResult {
	grouped := make(map[string][]ContradictionResult)

	for _, r := range results {
		key := r.File1 + " <-> " + r.File2
		grouped[key] = append(grouped[key], r)
	}

	return grouped
}

func filterPairs(pairs []StatementPair, minSimilarity float64) []StatementPair {
	filtered := make([]StatementPair, 0, len(pairs))

	for _, p := range pairs {
		if p.Similarity >= minSimilarity {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

func severityOrder(s Severity) int {
	switch s {
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	default:
		return 0
	}
}
