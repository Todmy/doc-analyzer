package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
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
}

// SemanticAxesRequest represents a request to set semantic axes
type SemanticAxesRequest struct {
	Words []string `json:"words"`
}

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

	// Convert to model statements for clustering and anomaly detection
	modelStatements := s.convertToModelStatements(statements)

	// Run clustering to get cluster assignments
	clusterResult := s.clusteringService.AutoCluster(modelStatements, 10)

	// Get anomaly scores
	anomalyResults := s.anomalyService.DetectAnomalies(modelStatements)
	anomalyScores := make(map[int]float64)
	for _, a := range anomalyResults {
		anomalyScores[a.Index] = a.Score
	}

	// Build visualization points
	points := make([]VisualizationPoint, len(statements))
	for i, stmt := range statements {
		// Get document filename
		doc, _ := s.documentRepo.GetByID(r.Context(), stmt.DocumentID)
		filename := ""
		if doc != nil {
			filename = doc.Filename
		}

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

	// Convert to model statements for clustering
	modelStatements := s.convertToModelStatements(statements)

	// Run clustering
	clusterResult := s.clusteringService.AutoCluster(modelStatements, 10)

	// Get anomaly scores
	anomalyResults := s.anomalyService.DetectAnomalies(modelStatements)
	anomalyScores := make(map[int]float64)
	for _, a := range anomalyResults {
		anomalyScores[a.Index] = a.Score
	}

	// Build visualization points
	points := make([]VisualizationPoint, len(statements))
	for i, stmt := range statements {
		doc, _ := s.documentRepo.GetByID(r.Context(), stmt.DocumentID)
		filename := ""
		if doc != nil {
			filename = doc.Filename
		}

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
