package similarity

import (
	"sort"
)

// SimilarPair represents a pair of similar items with their similarity score.
type SimilarPair struct {
	Idx1       int     // Index of first item
	Idx2       int     // Index of second item
	Similarity float64 // Similarity score (0-1)
}

const DefaultThreshold = 0.75

// FindSimilarPairs finds all pairs of embeddings with similarity above the threshold.
// It skips self-pairs (i,i) and duplicate pairs (i,j) and (j,i), only keeping (i,j) where i < j.
// Returns pairs sorted by similarity score in descending order.
func FindSimilarPairs(embeddings [][]float32, threshold float64) []SimilarPair {
	if len(embeddings) == 0 {
		return []SimilarPair{}
	}

	// Use default threshold if not specified
	if threshold <= 0 {
		threshold = DefaultThreshold
	}

	var pairs []SimilarPair

	// Only iterate upper triangle to avoid duplicates
	for i := 0; i < len(embeddings); i++ {
		for j := i + 1; j < len(embeddings); j++ {
			sim := CosineSimilarity(embeddings[i], embeddings[j])
			if sim >= threshold {
				pairs = append(pairs, SimilarPair{
					Idx1:       i,
					Idx2:       j,
					Similarity: sim,
				})
			}
		}
	}

	// Sort by similarity descending
	sort.Slice(pairs, func(a, b int) bool {
		return pairs[a].Similarity > pairs[b].Similarity
	})

	return pairs
}

// FindSimilarPairsFromMatrix finds similar pairs from a precomputed similarity matrix.
// This is more efficient when you already have the similarity matrix computed.
func FindSimilarPairsFromMatrix(matrix [][]float64, threshold float64) []SimilarPair {
	if len(matrix) == 0 {
		return []SimilarPair{}
	}

	// Use default threshold if not specified
	if threshold <= 0 {
		threshold = DefaultThreshold
	}

	var pairs []SimilarPair

	// Only iterate upper triangle to avoid duplicates and self-pairs
	for i := 0; i < len(matrix); i++ {
		for j := i + 1; j < len(matrix[i]); j++ {
			if matrix[i][j] >= threshold {
				pairs = append(pairs, SimilarPair{
					Idx1:       i,
					Idx2:       j,
					Similarity: matrix[i][j],
				})
			}
		}
	}

	// Sort by similarity descending
	sort.Slice(pairs, func(a, b int) bool {
		return pairs[a].Similarity > pairs[b].Similarity
	})

	return pairs
}

// TopKSimilar finds the top-k most similar pairs
func TopKSimilar(embeddings [][]float32, k int) []SimilarPair {
	if len(embeddings) == 0 || k <= 0 {
		return []SimilarPair{}
	}

	// Find all pairs with minimum threshold
	pairs := FindSimilarPairs(embeddings, 0)

	// Return top-k
	if k < len(pairs) {
		return pairs[:k]
	}
	return pairs
}
