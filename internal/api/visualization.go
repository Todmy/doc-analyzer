package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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

	// TODO: Fetch actual visualization data from database
	// For now, return stub response
	respondJSON(w, http.StatusOK, VisualizationResponse{
		Points:     []VisualizationPoint{},
		Clusters:   []ClusterInfo{},
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

	var req SemanticAxesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(req.Words) == 0 || len(req.Words) > 3 {
		respondError(w, http.StatusBadRequest, "provide 1-3 words for semantic axes")
		return
	}

	// TODO: Implement semantic axis projection
	// 1. Embed the words
	// 2. Find dimensions with max values for each word
	// 3. Project statements onto these dimensions
	// 4. Return reprojected coordinates

	respondJSON(w, http.StatusOK, VisualizationResponse{
		Points:     []VisualizationPoint{},
		Clusters:   []ClusterInfo{},
		Dimensions: len(req.Words),
		Method:     "semantic",
		AxisLabels: req.Words,
	})
}
