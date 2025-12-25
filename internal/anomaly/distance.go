package anomaly

import (
	"math"
	"sort"
)

// DistanceAnomalyDetector detects anomalies based on k-nearest neighbor distances
type DistanceAnomalyDetector struct{}

// NewDistanceAnomalyDetector creates a new distance-based anomaly detector
func NewDistanceAnomalyDetector() *DistanceAnomalyDetector {
	return &DistanceAnomalyDetector{}
}

// Detect computes anomaly scores based on average distance to k-nearest neighbors
// embeddings: slice of embedding vectors
// k: number of nearest neighbors to consider
// Returns: anomaly scores (0-1), where higher score = more anomalous
func (d *DistanceAnomalyDetector) Detect(embeddings [][]float32, k int) []float64 {
	n := len(embeddings)
	if n == 0 {
		return []float64{}
	}

	// Ensure k is valid
	if k <= 0 {
		k = 5 // default
	}
	if k >= n {
		k = n - 1
	}

	scores := make([]float64, n)

	// For each point, compute average distance to k-nearest neighbors
	for i := 0; i < n; i++ {
		distances := make([]float64, 0, n-1)

		// Compute distances to all other points
		for j := 0; j < n; j++ {
			if i != j {
				dist := euclideanDistance(embeddings[i], embeddings[j])
				distances = append(distances, dist)
			}
		}

		// Sort distances to find k-nearest neighbors
		sort.Float64s(distances)

		// Compute average distance to k-nearest neighbors
		avgDist := 0.0
		actualK := k
		if actualK > len(distances) {
			actualK = len(distances)
		}
		for ki := 0; ki < actualK; ki++ {
			avgDist += distances[ki]
		}
		if actualK > 0 {
			avgDist /= float64(actualK)
		}

		scores[i] = avgDist
	}

	// Normalize scores to 0-1 range
	return normalizeScores(scores)
}

// euclideanDistance computes the Euclidean distance between two vectors
func euclideanDistance(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	sum := 0.0
	for i := range a {
		diff := float64(a[i] - b[i])
		sum += diff * diff
	}

	return math.Sqrt(sum)
}

// normalizeScores normalizes scores to 0-1 range using min-max normalization
func normalizeScores(scores []float64) []float64 {
	if len(scores) == 0 {
		return scores
	}

	minScore := scores[0]
	maxScore := scores[0]

	for _, score := range scores {
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}
	}

	normalized := make([]float64, len(scores))
	scoreRange := maxScore - minScore

	if scoreRange == 0 {
		// All scores are the same
		for i := range normalized {
			normalized[i] = 0.5
		}
		return normalized
	}

	for i, score := range scores {
		normalized[i] = (score - minScore) / scoreRange
	}

	return normalized
}
