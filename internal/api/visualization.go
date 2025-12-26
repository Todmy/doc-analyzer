package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/todmy/doc-analyzer/internal/storage"
	"github.com/todmy/doc-analyzer/internal/visualization"
)

// VisualizationResponse represents the visualization data
type VisualizationResponse struct {
	Points     []VisualizationPoint `json:"points"`
	Clusters   []ClusterInfo        `json:"clusters"`
	Dimensions int                  `json:"dimensions"`
	Method     string               `json:"method"`
	AxisLabels []string             `json:"axis_labels,omitempty"`
}

// VisualizationPoint represents a point in the visualization
type VisualizationPoint struct {
	ID           string  `json:"id"`
	X            float64 `json:"x"`
	Y            float64 `json:"y"`
	Z            float64 `json:"z,omitempty"`
	ClusterID    int     `json:"cluster_id"`
	AnomalyScore float64 `json:"anomaly_score"`
	Preview      string  `json:"preview"`
	SourceFile   string  `json:"source_file"`
}

// ClusterInfo represents cluster metadata for visualization
type ClusterInfo struct {
	ID       int      `json:"id"`
	Keywords []string `json:"keywords"`
	Color    string   `json:"color"`
	Size     int      `json:"size"`
	Density  float64  `json:"density"`
}

// SemanticAxesRequest represents a request to set semantic axes
type SemanticAxesRequest struct {
	Words []string `json:"words"`
}

// maxVisualizationPoints is the maximum number of points to render for performance
// PCA/SVD is O(n*d²) so we limit to 1000 for acceptable response times
const maxVisualizationPoints = 1000

// handleGetVisualization returns visualization data for a project
func (s *Server) handleGetVisualizationImpl(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project id is required")
		return
	}

	pid, err := uuid.Parse(projectID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	// Parse dimensions parameter (default 2)
	dimensions := 2
	if d := r.URL.Query().Get("dimensions"); d == "3" {
		dimensions = 3
	}

	// Parse method parameter (default pca)
	method := r.URL.Query().Get("method")
	if method == "" {
		method = "pca"
	}

	// Parse words parameter for semantic method
	words := r.URL.Query()["words"]

	// Get statements for project
	statements, err := s.statementRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch statements")
		return
	}

	// Sample statements if too many for performance
	if len(statements) > maxVisualizationPoints {
		statements = sampleStatements(statements, maxVisualizationPoints)
	}

	// Pre-load documents to avoid N+1 queries
	docs, err := s.documentRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch documents")
		return
	}
	docMap := make(map[string]string, len(docs))
	for _, doc := range docs {
		docMap[doc.ID.String()] = doc.Filename
	}

	if len(statements) == 0 {
		respondJSON(w, http.StatusOK, VisualizationResponse{
			Points:     []VisualizationPoint{},
			Clusters:   []ClusterInfo{},
			Dimensions: dimensions,
			Method:     method,
		})
		return
	}

	// Extract embeddings
	embeddings := make([][]float32, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding.Slice()
	}

	// Get visualization coordinates
	visResult, err := s.visualizationService.GetVisualization(r.Context(), embeddings, method, dimensions, words)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate visualization")
		return
	}

	// Convert to model statements for anomaly detection
	modelStatements := s.convertToModelStatements(statements)

	// Run clustering on projected coordinates (much faster than full embeddings)
	coords := extractCoords(visResult.Points, dimensions)
	texts := extractTexts(statements)
	clusterResult := s.clusteringService.AutoClusterCoordinates(coords, texts, 10)

	// Get anomaly scores
	anomalyResults := s.anomalyService.DetectAnomalies(modelStatements)
	anomalyScores := make(map[int]float64)
	for _, a := range anomalyResults {
		anomalyScores[a.Index] = a.Score
	}

	// Build visualization points
	points := make([]VisualizationPoint, len(statements))
	for i, stmt := range statements {
		// Get document filename from pre-loaded map
		filename := docMap[stmt.DocumentID.String()]

		// Truncate text for preview
		preview := stmt.Text
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		points[i] = VisualizationPoint{
			ID:           stmt.ID.String(),
			X:            visResult.Points[i].X,
			Y:            visResult.Points[i].Y,
			Z:            visResult.Points[i].Z,
			ClusterID:    clusterResult.Labels[i],
			AnomalyScore: anomalyScores[i],
			Preview:      preview,
			SourceFile:   filename,
		}
	}

	// Build cluster info
	clusterColors := []string{"#3498db", "#e74c3c", "#2ecc71", "#f39c12", "#9b59b6", "#1abc9c", "#e91e63", "#00bcd4", "#ff5722", "#607d8b"}
	clusters := make([]ClusterInfo, len(clusterResult.Clusters))
	for i, c := range clusterResult.Clusters {
		keywords := make([]string, len(c.Keywords))
		for j, kw := range c.Keywords {
			keywords[j] = kw.Word
		}
		color := clusterColors[i%len(clusterColors)]
		clusters[i] = ClusterInfo{
			ID:       c.ID,
			Keywords: keywords,
			Color:    color,
			Size:     c.Size,
			Density:  c.Density,
		}
	}

	respondJSON(w, http.StatusOK, VisualizationResponse{
		Points:     points,
		Clusters:   clusters,
		Dimensions: dimensions,
		Method:     method,
	})
}

// handleSetAxes sets semantic axes for visualization
func (s *Server) handleSetAxesImpl(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project id is required")
		return
	}

	pid, err := uuid.Parse(projectID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var req SemanticAxesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Words) == 0 || len(req.Words) > 3 {
		respondError(w, http.StatusBadRequest, "provide 1-3 words for semantic axes")
		return
	}

	// Check if embedding client is configured for semantic axes
	if s.embeddingClient == nil {
		respondError(w, http.StatusServiceUnavailable, "embedding service not configured - set OPENROUTER_API_KEY")
		return
	}

	// Get statements for project
	statements, err := s.statementRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch statements")
		return
	}

	// Sample statements if too many for performance
	if len(statements) > maxVisualizationPoints {
		statements = sampleStatements(statements, maxVisualizationPoints)
	}

	// Pre-load documents to avoid N+1 queries
	docs, err := s.documentRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch documents")
		return
	}
	docMap := make(map[string]string, len(docs))
	for _, doc := range docs {
		docMap[doc.ID.String()] = doc.Filename
	}

	if len(statements) == 0 {
		respondJSON(w, http.StatusOK, VisualizationResponse{
			Points:     []VisualizationPoint{},
			Clusters:   []ClusterInfo{},
			Dimensions: len(req.Words),
			Method:     "semantic",
			AxisLabels: req.Words,
		})
		return
	}

	// Extract embeddings
	embeddings := make([][]float32, len(statements))
	for i, stmt := range statements {
		embeddings[i] = stmt.Embedding.Slice()
	}

	// Get visualization coordinates using semantic axes
	visResult, err := s.visualizationService.GetVisualization(r.Context(), embeddings, "semantic", len(req.Words), req.Words)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to generate semantic visualization: "+err.Error())
		return
	}

	// Convert to model statements for anomaly detection
	modelStatements := s.convertToModelStatements(statements)

	// Run clustering on projected coordinates (semantic mode)
	coords := extractCoords(visResult.Points, len(req.Words))
	texts := extractTexts(statements)
	clusterResult := s.clusteringService.AutoClusterCoordinates(coords, texts, 10)

	// Get anomaly scores
	anomalyResults := s.anomalyService.DetectAnomalies(modelStatements)
	anomalyScores := make(map[int]float64)
	for _, a := range anomalyResults {
		anomalyScores[a.Index] = a.Score
	}

	// Build visualization points
	points := make([]VisualizationPoint, len(statements))
	for i, stmt := range statements {
		// Get document filename from pre-loaded map
		filename := docMap[stmt.DocumentID.String()]

		preview := stmt.Text
		if len(preview) > 100 {
			preview = preview[:100] + "..."
		}

		points[i] = VisualizationPoint{
			ID:           stmt.ID.String(),
			X:            visResult.Points[i].X,
			Y:            visResult.Points[i].Y,
			Z:            visResult.Points[i].Z,
			ClusterID:    clusterResult.Labels[i],
			AnomalyScore: anomalyScores[i],
			Preview:      preview,
			SourceFile:   filename,
		}
	}

	// Build cluster info
	clusterColors := []string{"#3498db", "#e74c3c", "#2ecc71", "#f39c12", "#9b59b6", "#1abc9c", "#e91e63", "#00bcd4", "#ff5722", "#607d8b"}
	clusters := make([]ClusterInfo, len(clusterResult.Clusters))
	for i, c := range clusterResult.Clusters {
		keywords := make([]string, len(c.Keywords))
		for j, kw := range c.Keywords {
			keywords[j] = kw.Word
		}
		color := clusterColors[i%len(clusterColors)]
		clusters[i] = ClusterInfo{
			ID:       c.ID,
			Keywords: keywords,
			Color:    color,
			Size:     c.Size,
			Density:  c.Density,
		}
	}

	respondJSON(w, http.StatusOK, VisualizationResponse{
		Points:     points,
		Clusters:   clusters,
		Dimensions: len(req.Words),
		Method:     "semantic",
		AxisLabels: req.Words,
	})
}

// extractCoords extracts 2D or 3D coordinates from visualization points
func extractCoords(points []visualization.Point, dimensions int) [][]float64 {
	coords := make([][]float64, len(points))
	for i, p := range points {
		if dimensions == 3 {
			coords[i] = []float64{p.X, p.Y, p.Z}
		} else {
			coords[i] = []float64{p.X, p.Y}
		}
	}
	return coords
}

// extractTexts extracts text content from statements
func extractTexts(statements []*storage.Statement) []string {
	texts := make([]string, len(statements))
	for i, stmt := range statements {
		texts[i] = stmt.Text
	}
	return texts
}

// sampleStatements returns a uniformly distributed sample of statements
func sampleStatements(statements []*storage.Statement, maxCount int) []*storage.Statement {
	if len(statements) <= maxCount {
		return statements
	}

	// Use deterministic sampling by taking evenly spaced indices
	step := float64(len(statements)) / float64(maxCount)
	sampled := make([]*storage.Statement, maxCount)
	for i := 0; i < maxCount; i++ {
		idx := int(float64(i) * step)
		sampled[i] = statements[idx]
	}
	return sampled
}
