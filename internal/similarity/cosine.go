package similarity

import (
	"math"

	"gonum.org/v1/gonum/floats"
)

// CosineSimilarity calculates the cosine similarity between two vectors.
// Returns a value between -1 and 1, where 1 means identical direction,
// 0 means orthogonal, and -1 means opposite direction.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	if len(a) == 0 {
		return 0
	}

	// Convert float32 slices to float64 for gonum
	aFloat64 := make([]float64, len(a))
	bFloat64 := make([]float64, len(b))
	for i := range a {
		aFloat64[i] = float64(a[i])
		bFloat64[i] = float64(b[i])
	}

	// Calculate dot product
	dotProduct := floats.Dot(aFloat64, bFloat64)

	// Calculate magnitudes
	magA := math.Sqrt(floats.Dot(aFloat64, aFloat64))
	magB := math.Sqrt(floats.Dot(bFloat64, bFloat64))

	// Avoid division by zero
	if magA == 0 || magB == 0 {
		return 0
	}

	return dotProduct / (magA * magB)
}

// CosineSimilarityMatrix calculates pairwise cosine similarity for all embeddings.
// Returns an n×n matrix where element [i][j] is the similarity between embeddings[i] and embeddings[j].
// The matrix is symmetric and diagonal elements are 1.0 (self-similarity).
func CosineSimilarityMatrix(embeddings [][]float32) [][]float64 {
	n := len(embeddings)
	if n == 0 {
		return [][]float64{}
	}

	// Initialize matrix
	matrix := make([][]float64, n)
	for i := range matrix {
		matrix[i] = make([]float64, n)
	}

	// Calculate similarities
	// Only compute upper triangle since matrix is symmetric
	for i := 0; i < n; i++ {
		matrix[i][i] = 1.0 // Self-similarity is always 1
		for j := i + 1; j < n; j++ {
			sim := CosineSimilarity(embeddings[i], embeddings[j])
			matrix[i][j] = sim
			matrix[j][i] = sim // Symmetric
		}
	}

	return matrix
}

// CosineDistance calculates the cosine distance between two vectors.
// Distance = 1 - similarity, returns value between 0 and 2.
func CosineDistance(a, b []float32) float64 {
	return 1 - CosineSimilarity(a, b)
}
