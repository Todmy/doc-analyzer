package similarity

import (
	"github.com/todmy/doc-analyzer/pkg/models"
)

// Service provides similarity analysis functionality.
type Service struct {
	threshold float64
}

// NewService creates a new similarity service with the specified threshold.
// If threshold is 0 or negative, uses DefaultThreshold (0.75).
func NewService(threshold float64) *Service {
	if threshold <= 0 {
		threshold = DefaultThreshold
	}
	return &Service{
		threshold: threshold,
	}
}

// SimilarPairResult contains detailed information about a similar pair of statements.
type SimilarPairResult struct {
	Statement1  string  `json:"statement1"`
	Statement2  string  `json:"statement2"`
	File1       string  `json:"file1"`
	File2       string  `json:"file2"`
	Line1       int     `json:"line1"`
	Line2       int     `json:"line2"`
	Similarity  float64 `json:"similarity"`
	Index1      int     `json:"index1"`
	Index2      int     `json:"index2"`
}

// FindSimilarStatements finds similar statement pairs from a list of statements.
// Returns detailed results including statement text, file info, and similarity scores.
func (s *Service) FindSimilarStatements(statements []models.Statement, threshold float64) []SimilarPairResult {
	if len(statements) == 0 {
		return []SimilarPairResult{}
	}

	// Use service threshold if not specified
	if threshold <= 0 {
		threshold = s.threshold
	}

	// Extract embeddings from statements
	embeddings := make([][]float32, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding
	}

	// Find similar pairs
	pairs := FindSimilarPairs(embeddings, threshold)

	// Convert to detailed results
	results := make([]SimilarPairResult, len(pairs))
	for i, pair := range pairs {
		stmt1 := statements[pair.Idx1]
		stmt2 := statements[pair.Idx2]

		results[i] = SimilarPairResult{
			Statement1: stmt1.Text,
			Statement2: stmt2.Text,
			File1:      stmt1.File,
			File2:      stmt2.File,
			Line1:      stmt1.Line,
			Line2:      stmt2.Line,
			Similarity: pair.Similarity,
			Index1:     pair.Idx1,
			Index2:     pair.Idx2,
		}
	}

	return results
}

// FindSimilarStatementsWithMatrix is an optimized version that uses a precomputed similarity matrix.
// Use this when you need to find similar pairs with multiple different thresholds.
func (s *Service) FindSimilarStatementsWithMatrix(statements []models.Statement, matrix [][]float64, threshold float64) []SimilarPairResult {
	if len(statements) == 0 || len(matrix) == 0 {
		return []SimilarPairResult{}
	}

	// Use service threshold if not specified
	if threshold <= 0 {
		threshold = s.threshold
	}

	// Find similar pairs from matrix
	pairs := FindSimilarPairsFromMatrix(matrix, threshold)

	// Convert to detailed results
	results := make([]SimilarPairResult, len(pairs))
	for i, pair := range pairs {
		stmt1 := statements[pair.Idx1]
		stmt2 := statements[pair.Idx2]

		results[i] = SimilarPairResult{
			Statement1: stmt1.Text,
			Statement2: stmt2.Text,
			File1:      stmt1.File,
			File2:      stmt2.File,
			Line1:      stmt1.Line,
			Line2:      stmt2.Line,
			Similarity: pair.Similarity,
			Index1:     pair.Idx1,
			Index2:     pair.Idx2,
		}
	}

	return results
}

// SetThreshold updates the default threshold for the service.
func (s *Service) SetThreshold(threshold float64) {
	if threshold > 0 {
		s.threshold = threshold
	}
}

// GetThreshold returns the current default threshold.
func (s *Service) GetThreshold() float64 {
	return s.threshold
}

// ComputeSimilarityMatrix computes and returns the full similarity matrix for statements
func (s *Service) ComputeSimilarityMatrix(statements []models.Statement) [][]float64 {
	if len(statements) == 0 {
		return [][]float64{}
	}

	embeddings := make([][]float32, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding
	}

	return CosineSimilarityMatrix(embeddings)
}
