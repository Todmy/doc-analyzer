package visualization

import (
	"fmt"
	"math"

	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
)

// Reducer defines the interface for dimensionality reduction
type Reducer interface {
	Reduce(embeddings [][]float32, dims int) ([][]float64, error)
	Name() string
}

// PCAReducer implements PCA dimensionality reduction
type PCAReducer struct{}

// NewPCAReducer creates a new PCA reducer
func NewPCAReducer() *PCAReducer {
	return &PCAReducer{}
}

// Name returns the reducer name
func (r *PCAReducer) Name() string {
	return "pca"
}

// Reduce performs PCA dimensionality reduction
func (r *PCAReducer) Reduce(embeddings [][]float32, dims int) ([][]float64, error) {
	if len(embeddings) == 0 {
		return nil, nil
	}

	n := len(embeddings)
	d := len(embeddings[0])

	if dims > d {
		dims = d
	}
	if dims > n {
		dims = n
	}

	// Convert to float64 matrix
	data := make([]float64, n*d)
	for i, emb := range embeddings {
		for j, v := range emb {
			data[i*d+j] = float64(v)
		}
	}
	X := mat.NewDense(n, d, data)

	// Center the data
	means := make([]float64, d)
	for j := 0; j < d; j++ {
		col := mat.Col(nil, j, X)
		means[j] = stat.Mean(col, nil)
	}

	centered := mat.NewDense(n, d, nil)
	for i := 0; i < n; i++ {
		for j := 0; j < d; j++ {
			centered.Set(i, j, X.At(i, j)-means[j])
		}
	}

	// Compute SVD
	var svd mat.SVD
	ok := svd.Factorize(centered, mat.SVDThin)
	if !ok {
		return nil, fmt.Errorf("SVD factorization failed")
	}

	// Get V matrix (right singular vectors)
	var v mat.Dense
	svd.VTo(&v)

	// Project onto first dims components
	vReduced := v.Slice(0, d, 0, dims).(*mat.Dense)
	result := mat.NewDense(n, dims, nil)
	result.Mul(centered, vReduced)

	// Convert back to [][]float64
	reduced := make([][]float64, n)
	for i := 0; i < n; i++ {
		reduced[i] = make([]float64, dims)
		for j := 0; j < dims; j++ {
			reduced[i][j] = result.At(i, j)
		}
	}

	// Normalize to [-1, 1] range for visualization
	reduced = normalizeCoordinates(reduced)

	return reduced, nil
}

// normalizeCoordinates scales coordinates to [-1, 1] range
func normalizeCoordinates(coords [][]float64) [][]float64 {
	if len(coords) == 0 {
		return coords
	}

	dims := len(coords[0])
	mins := make([]float64, dims)
	maxs := make([]float64, dims)

	for j := 0; j < dims; j++ {
		mins[j] = math.MaxFloat64
		maxs[j] = -math.MaxFloat64
	}

	for _, coord := range coords {
		for j, v := range coord {
			if v < mins[j] {
				mins[j] = v
			}
			if v > maxs[j] {
				maxs[j] = v
			}
		}
	}

	normalized := make([][]float64, len(coords))
	for i, coord := range coords {
		normalized[i] = make([]float64, dims)
		for j, v := range coord {
			rng := maxs[j] - mins[j]
			if rng == 0 {
				normalized[i][j] = 0
			} else {
				normalized[i][j] = 2*(v-mins[j])/rng - 1
			}
		}
	}

	return normalized
}
