package clustering

import (
	"math"
	"math/rand"

	"gonum.org/v1/gonum/floats"
)

// KMeans performs k-means clustering on embeddings
type KMeans struct {
	K          int       // Number of clusters
	MaxIter    int       // Maximum iterations
	Tolerance  float64   // Convergence tolerance
	Centroids  [][]float64
	Labels     []int
	Inertia    float64
}

// NewKMeans creates a new K-means clusterer
func NewKMeans(k int) *KMeans {
	return &KMeans{
		K:         k,
		MaxIter:   100,
		Tolerance: 1e-4,
	}
}

// Fit clusters the embeddings and returns cluster assignments
func (km *KMeans) Fit(embeddings [][]float32) []int {
	n := len(embeddings)
	if n == 0 || km.K <= 0 {
		return []int{}
	}

	// Adjust k if necessary
	k := km.K
	if k > n {
		k = n
	}

	// Convert embeddings to float64
	data := make([][]float64, n)
	for i, e := range embeddings {
		data[i] = make([]float64, len(e))
		for j, v := range e {
			data[i][j] = float64(v)
		}
	}

	dim := len(data[0])

	// Initialize centroids using k-means++ algorithm
	km.Centroids = kMeansPlusPlusInit(data, k)

	km.Labels = make([]int, n)
	var prevInertia float64

	for iter := 0; iter < km.MaxIter; iter++ {
		// Assign points to nearest centroid
		km.Inertia = 0
		for i, point := range data {
			minDist := math.MaxFloat64
			minIdx := 0
			for j, centroid := range km.Centroids {
				dist := squaredEuclideanDistance(point, centroid)
				if dist < minDist {
					minDist = dist
					minIdx = j
				}
			}
			km.Labels[i] = minIdx
			km.Inertia += minDist
		}

		// Check convergence
		if iter > 0 && math.Abs(prevInertia-km.Inertia) < km.Tolerance {
			break
		}
		prevInertia = km.Inertia

		// Update centroids
		counts := make([]int, k)
		newCentroids := make([][]float64, k)
		for i := range newCentroids {
			newCentroids[i] = make([]float64, dim)
		}

		for i, label := range km.Labels {
			counts[label]++
			floats.Add(newCentroids[label], data[i])
		}

		for i := range newCentroids {
			if counts[i] > 0 {
				floats.Scale(1.0/float64(counts[i]), newCentroids[i])
			}
		}
		km.Centroids = newCentroids
	}

	return km.Labels
}

// Predict assigns new points to the nearest cluster
func (km *KMeans) Predict(embeddings [][]float32) []int {
	if len(km.Centroids) == 0 {
		return []int{}
	}

	labels := make([]int, len(embeddings))
	for i, e := range embeddings {
		point := make([]float64, len(e))
		for j, v := range e {
			point[j] = float64(v)
		}

		minDist := math.MaxFloat64
		minIdx := 0
		for j, centroid := range km.Centroids {
			dist := squaredEuclideanDistance(point, centroid)
			if dist < minDist {
				minDist = dist
				minIdx = j
			}
		}
		labels[i] = minIdx
	}

	return labels
}

// GetCentroids returns the cluster centroids as float32
func (km *KMeans) GetCentroids() [][]float32 {
	result := make([][]float32, len(km.Centroids))
	for i, c := range km.Centroids {
		result[i] = make([]float32, len(c))
		for j, v := range c {
			result[i][j] = float32(v)
		}
	}
	return result
}

// kMeansPlusPlusInit initializes centroids using k-means++ algorithm
func kMeansPlusPlusInit(data [][]float64, k int) [][]float64 {
	n := len(data)
	centroids := make([][]float64, 0, k)

	// Create deterministic seed based on data to ensure reproducible results
	seed := computeDataSeed(data)
	rng := rand.New(rand.NewSource(seed))

	// Choose first centroid randomly
	firstIdx := rng.Intn(n)
	centroids = append(centroids, copySlice(data[firstIdx]))

	// Choose remaining centroids with probability proportional to distance squared
	distances := make([]float64, n)
	for i := 1; i < k; i++ {
		// Update distances to nearest centroid
		totalDist := 0.0
		for j, point := range data {
			minDist := math.MaxFloat64
			for _, centroid := range centroids {
				dist := squaredEuclideanDistance(point, centroid)
				if dist < minDist {
					minDist = dist
				}
			}
			distances[j] = minDist
			totalDist += minDist
		}

		// Choose next centroid with probability proportional to distance squared
		r := rng.Float64() * totalDist
		cumSum := 0.0
		for j, d := range distances {
			cumSum += d
			if cumSum >= r {
				centroids = append(centroids, copySlice(data[j]))
				break
			}
		}
	}

	return centroids
}

// computeDataSeed creates a deterministic seed from the data
func computeDataSeed(data [][]float64) int64 {
	if len(data) == 0 {
		return 42
	}
	// Use a simple hash based on data size and sample values
	seed := int64(len(data))
	if len(data[0]) > 0 {
		seed += int64(len(data[0])) * 1000
		// Sample some values to create variation
		seed += int64(data[0][0] * 1000000)
		if len(data) > 1 {
			seed += int64(data[len(data)/2][0] * 1000000)
		}
		if len(data) > 2 {
			seed += int64(data[len(data)-1][0] * 1000000)
		}
	}
	return seed
}

func squaredEuclideanDistance(a, b []float64) float64 {
	sum := 0.0
	for i := range a {
		diff := a[i] - b[i]
		sum += diff * diff
	}
	return sum
}

func copySlice(s []float64) []float64 {
	result := make([]float64, len(s))
	copy(result, s)
	return result
}

// ElbowMethod helps find optimal k using the elbow method
func ElbowMethod(embeddings [][]float32, maxK int) []float64 {
	if maxK <= 0 {
		maxK = 10
	}
	if maxK > len(embeddings) {
		maxK = len(embeddings)
	}

	inertias := make([]float64, maxK)
	for k := 1; k <= maxK; k++ {
		km := NewKMeans(k)
		km.Fit(embeddings)
		inertias[k-1] = km.Inertia
	}

	return inertias
}
