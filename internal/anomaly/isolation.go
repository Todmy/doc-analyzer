package anomaly

import (
	"math"
	"math/rand"
)

// IsolationForest implements a simplified isolation forest for anomaly detection
type IsolationForest struct {
	Trees      []*IsolationTree
	NumTrees   int
	SampleSize int
}

// IsolationTree represents a single tree in the forest
type IsolationTree struct {
	Root *IsolationNode
}

// IsolationNode represents a node in an isolation tree
type IsolationNode struct {
	SplitFeature int
	SplitValue   float64
	Left         *IsolationNode
	Right        *IsolationNode
	Size         int // Size of data that reached this node (for external nodes)
}

// NewIsolationForest creates a new isolation forest
func NewIsolationForest(numTrees, sampleSize int) *IsolationForest {
	if numTrees <= 0 {
		numTrees = 100
	}
	if sampleSize <= 0 {
		sampleSize = 256
	}

	return &IsolationForest{
		NumTrees:   numTrees,
		SampleSize: sampleSize,
	}
}

// Fit builds the isolation forest from the data
func (f *IsolationForest) Fit(data [][]float32) {
	n := len(data)
	if n == 0 {
		return
	}

	sampleSize := f.SampleSize
	if sampleSize > n {
		sampleSize = n
	}

	maxDepth := int(math.Ceil(math.Log2(float64(sampleSize))))

	f.Trees = make([]*IsolationTree, f.NumTrees)
	for i := 0; i < f.NumTrees; i++ {
		// Sample without replacement
		sample := sampleData(data, sampleSize)
		f.Trees[i] = &IsolationTree{
			Root: buildIsolationTree(sample, 0, maxDepth),
		}
	}
}

// Score returns anomaly scores for each point (higher = more anomalous)
func (f *IsolationForest) Score(data [][]float32) []float64 {
	n := len(data)
	if n == 0 || len(f.Trees) == 0 {
		return []float64{}
	}

	scores := make([]float64, n)

	// Expected path length for unsuccessful search in BST
	c := expectedPathLength(float64(f.SampleSize))

	for i, point := range data {
		// Average path length across all trees
		avgPathLength := 0.0
		for _, tree := range f.Trees {
			avgPathLength += pathLength(point, tree.Root, 0)
		}
		avgPathLength /= float64(len(f.Trees))

		// Anomaly score: 2^(-avgPathLength/c)
		// Higher score = more anomalous (shorter path = more isolated)
		scores[i] = math.Pow(2, -avgPathLength/c)
	}

	return scores
}

// buildIsolationTree recursively builds an isolation tree
func buildIsolationTree(data [][]float32, depth, maxDepth int) *IsolationNode {
	n := len(data)

	// Terminal conditions
	if n <= 1 || depth >= maxDepth {
		return &IsolationNode{Size: n}
	}

	// Pick random feature
	numFeatures := len(data[0])
	if numFeatures == 0 {
		return &IsolationNode{Size: n}
	}
	feature := rand.Intn(numFeatures)

	// Find min/max for this feature
	minVal := float64(data[0][feature])
	maxVal := float64(data[0][feature])
	for _, point := range data {
		v := float64(point[feature])
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// If all values are the same, can't split
	if minVal == maxVal {
		return &IsolationNode{Size: n}
	}

	// Random split value
	splitValue := minVal + rand.Float64()*(maxVal-minVal)

	// Partition data
	var left, right [][]float32
	for _, point := range data {
		if float64(point[feature]) < splitValue {
			left = append(left, point)
		} else {
			right = append(right, point)
		}
	}

	// Handle edge case where all points go to one side
	if len(left) == 0 || len(right) == 0 {
		return &IsolationNode{Size: n}
	}

	return &IsolationNode{
		SplitFeature: feature,
		SplitValue:   splitValue,
		Left:         buildIsolationTree(left, depth+1, maxDepth),
		Right:        buildIsolationTree(right, depth+1, maxDepth),
	}
}

// pathLength computes the path length for a point in the tree
func pathLength(point []float32, node *IsolationNode, currentDepth int) float64 {
	if node == nil {
		return float64(currentDepth)
	}

	// External node
	if node.Left == nil && node.Right == nil {
		return float64(currentDepth) + expectedPathLength(float64(node.Size))
	}

	// Internal node - follow the path
	if float64(point[node.SplitFeature]) < node.SplitValue {
		return pathLength(point, node.Left, currentDepth+1)
	}
	return pathLength(point, node.Right, currentDepth+1)
}

// expectedPathLength computes the expected path length for unsuccessful search in BST
func expectedPathLength(n float64) float64 {
	if n <= 1 {
		return 0
	}
	if n <= 2 {
		return 1
	}
	// c(n) = 2H(n-1) - 2(n-1)/n
	// H(i) is the harmonic number ≈ ln(i) + euler_gamma
	return 2.0*(math.Log(n-1)+0.5772156649) - 2.0*(n-1)/n
}

// sampleData samples data without replacement
func sampleData(data [][]float32, sampleSize int) [][]float32 {
	n := len(data)
	if sampleSize >= n {
		result := make([][]float32, n)
		copy(result, data)
		return result
	}

	// Fisher-Yates shuffle for sampling
	indices := make([]int, n)
	for i := range indices {
		indices[i] = i
	}
	for i := 0; i < sampleSize; i++ {
		j := i + rand.Intn(n-i)
		indices[i], indices[j] = indices[j], indices[i]
	}

	result := make([][]float32, sampleSize)
	for i := 0; i < sampleSize; i++ {
		result[i] = data[indices[i]]
	}
	return result
}
