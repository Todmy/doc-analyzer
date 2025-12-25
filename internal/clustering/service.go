package clustering

import (
	"github.com/todmy/doc-analyzer/pkg/models"
)

// Service provides clustering functionality
type Service struct {
	keywordExtractor *KeywordExtractor
	defaultK         int
	keywordsPerCluster int
}

// Config holds clustering service configuration
type Config struct {
	DefaultK           int
	KeywordsPerCluster int
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		DefaultK:           5,
		KeywordsPerCluster: 5,
	}
}

// NewService creates a new clustering service
func NewService(config Config) *Service {
	if config.DefaultK <= 0 {
		config.DefaultK = DefaultConfig().DefaultK
	}
	if config.KeywordsPerCluster <= 0 {
		config.KeywordsPerCluster = DefaultConfig().KeywordsPerCluster
	}

	return &Service{
		keywordExtractor:   NewKeywordExtractor(),
		defaultK:           config.DefaultK,
		keywordsPerCluster: config.KeywordsPerCluster,
	}
}

// ClusterResult represents the result of clustering
type ClusterResult struct {
	Clusters []Cluster
	Labels   []int
	K        int
	Inertia  float64
}

// Cluster represents a single cluster with its metadata
type Cluster struct {
	ID        int
	Centroid  []float32
	Size      int
	Keywords  []Keyword
	Density   float64
}

// ClusterStatements clusters statements and returns detailed results
func (s *Service) ClusterStatements(statements []models.Statement, k int) *ClusterResult {
	if len(statements) == 0 {
		return &ClusterResult{}
	}

	if k <= 0 {
		k = s.defaultK
	}
	if k > len(statements) {
		k = len(statements)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(statements))
	texts := make([]string, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding
		texts[i] = stmt.Text
	}

	// Run K-means
	km := NewKMeans(k)
	labels := km.Fit(embeddings)

	// Extract keywords for each cluster
	clusterKeywords := s.keywordExtractor.ExtractClusterKeywords(texts, labels, k, s.keywordsPerCluster)

	// Build cluster metadata
	clusters := make([]Cluster, k)
	clusterSizes := make([]int, k)
	for _, label := range labels {
		clusterSizes[label]++
	}

	centroids := km.GetCentroids()
	for i := 0; i < k; i++ {
		clusters[i] = Cluster{
			ID:       i,
			Centroid: centroids[i],
			Size:     clusterSizes[i],
			Keywords: clusterKeywords[i],
			Density:  s.computeDensity(embeddings, labels, i, centroids[i]),
		}
	}

	return &ClusterResult{
		Clusters: clusters,
		Labels:   labels,
		K:        k,
		Inertia:  km.Inertia,
	}
}

// AutoCluster determines optimal k using elbow method
func (s *Service) AutoCluster(statements []models.Statement, maxK int) *ClusterResult {
	if len(statements) == 0 {
		return &ClusterResult{}
	}

	if maxK <= 0 {
		maxK = 10
	}
	if maxK > len(statements) {
		maxK = len(statements)
	}

	// Extract embeddings
	embeddings := make([][]float32, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding
	}

	// Find optimal k using elbow method
	inertias := ElbowMethod(embeddings, maxK)
	optimalK := findElbow(inertias)

	return s.ClusterStatements(statements, optimalK)
}

// computeDensity calculates the average distance of cluster members to centroid
func (s *Service) computeDensity(embeddings [][]float32, labels []int, clusterID int, centroid []float32) float64 {
	totalDist := 0.0
	count := 0

	for i, label := range labels {
		if label == clusterID {
			dist := 0.0
			for j := range embeddings[i] {
				diff := float64(embeddings[i][j] - centroid[j])
				dist += diff * diff
			}
			totalDist += dist
			count++
		}
	}

	if count == 0 {
		return 0
	}

	// Return inverse of average distance (higher = denser)
	avgDist := totalDist / float64(count)
	if avgDist == 0 {
		return 1.0
	}
	return 1.0 / avgDist
}

// findElbow finds the elbow point in inertia curve
func findElbow(inertias []float64) int {
	if len(inertias) <= 2 {
		return len(inertias)
	}

	// Use the "knee" detection algorithm
	// Find point with maximum distance to line from first to last point
	n := len(inertias)

	// Line from first to last point
	x1, y1 := 0.0, inertias[0]
	x2, y2 := float64(n-1), inertias[n-1]

	maxDist := 0.0
	elbow := 1

	// Normalize coordinates
	xRange := x2 - x1
	yRange := y1 - y2 // y1 is typically larger

	for i := 1; i < n-1; i++ {
		// Point on curve
		x0 := float64(i) / xRange
		y0 := (inertias[i] - y2) / yRange

		// Normalized line endpoints
		nx1, ny1 := 0.0, 1.0
		nx2, ny2 := 1.0, 0.0

		// Distance from point to line
		num := abs((ny2-ny1)*x0 - (nx2-nx1)*y0 + nx2*ny1 - ny2*nx1)
		den := sqrt((ny2-ny1)*(ny2-ny1) + (nx2-nx1)*(nx2-nx1))

		dist := num / den
		if dist > maxDist {
			maxDist = dist
			elbow = i + 1 // k is 1-indexed
		}
	}

	return elbow
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Newton's method
	z := x
	for i := 0; i < 10; i++ {
		z -= (z*z - x) / (2 * z)
	}
	return z
}
