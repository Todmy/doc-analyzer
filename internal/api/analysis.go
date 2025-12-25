package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// AnalysisRequest represents a request to start analysis
type AnalysisRequest struct {
	ProjectID string `json:"project_id"`
}

// AnalysisStatusResponse represents the analysis status
type AnalysisStatusResponse struct {
	ProjectID string `json:"project_id"`
	Status    string `json:"status"`
	Progress  int    `json:"progress"`
}

// ClusterResponse represents a cluster in the API response
type ClusterResponse struct {
	ID       int      `json:"id"`
	Keywords []string `json:"keywords"`
	Size     int      `json:"size"`
	Density  float64  `json:"density"`
}

// SimilarPairResponse represents a similar pair in the API response
type SimilarPairResponse struct {
	Statement1 string  `json:"statement1"`
	Statement2 string  `json:"statement2"`
	File1      string  `json:"file1"`
	File2      string  `json:"file2"`
	Similarity float64 `json:"similarity"`
}

// AnomalyResponse represents an anomaly in the API response
type AnomalyResponse struct {
	Text       string  `json:"text"`
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Score      float64 `json:"score"`
}

// ContradictionResponse represents a contradiction in the API response
type ContradictionResponse struct {
	Statement1  string  `json:"statement1"`
	Statement2  string  `json:"statement2"`
	File1       string  `json:"file1"`
	File2       string  `json:"file2"`
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Explanation string  `json:"explanation"`
	Confidence  float64 `json:"confidence"`
}

// handleAnalyze starts the analysis pipeline for a project
func (s *Server) handleAnalyzeImpl(w http.ResponseWriter, r *http.Request) {
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

	// Verify project exists and user has access
	project, err := s.projectRepo.GetByID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch project")
		return
	}

	if project == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	userID := getUserIDFromContext(r.Context())
	if project.UserID.String() != userID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}

	// TODO: Start background analysis job
	// For now, return a stub response
	respondJSON(w, http.StatusAccepted, AnalysisStatusResponse{
		ProjectID: projectID,
		Status:    "queued",
		Progress:  0,
	})
}

// handleGetClusters returns clustering results for a project
func (s *Server) handleGetClustersImpl(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project id is required")
		return
	}

	// TODO: Fetch actual cluster data from database
	// For now, return empty array
	respondJSON(w, http.StatusOK, []ClusterResponse{})
}

// handleGetSimilarPairs returns similar pairs for a project
func (s *Server) handleGetSimilarPairsImpl(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project id is required")
		return
	}

	// Parse optional threshold parameter
	threshold := 0.75
	if t := r.URL.Query().Get("threshold"); t != "" {
		if _, err := json.Number(t).Float64(); err == nil {
			// Use the parsed threshold
		}
	}
	_ = threshold

	// TODO: Fetch actual similar pairs from database
	respondJSON(w, http.StatusOK, []SimilarPairResponse{})
}

// handleGetAnomalies returns anomaly detection results for a project
func (s *Server) handleGetAnomaliesImpl(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project id is required")
		return
	}

	// TODO: Fetch actual anomalies from database
	respondJSON(w, http.StatusOK, []AnomalyResponse{})
}

// handleGetContradictions returns contradiction detection results for a project
func (s *Server) handleGetContradictionsImpl(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	if projectID == "" {
		respondError(w, http.StatusBadRequest, "project id is required")
		return
	}

	// TODO: Fetch actual contradictions from database
	respondJSON(w, http.StatusOK, []ContradictionResponse{})
}
