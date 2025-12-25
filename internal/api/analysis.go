package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/todmy/doc-analyzer/internal/auth"
	"github.com/todmy/doc-analyzer/internal/clustering"
	"github.com/todmy/doc-analyzer/internal/contradiction"
	"github.com/todmy/doc-analyzer/internal/storage"
	"github.com/todmy/doc-analyzer/pkg/models"
)

// convertToModelStatements converts storage statements to model statements
func (s *Server) convertToModelStatements(statements []*storage.Statement) []models.Statement {
	result := make([]models.Statement, len(statements))
	for i, stmt := range statements {
		// Get document filename for source file
		doc, _ := s.documentRepo.GetByID(nil, stmt.DocumentID)
		filename := ""
		if doc != nil {
			filename = doc.Filename
		}

		result[i] = models.Statement{
			ID:         stmt.ID.String(),
			DocumentID: stmt.DocumentID.String(),
			Text:       stmt.Text,
			Position:   stmt.Position,
			Line:       stmt.Line,
			File:       filename,
			Embedding:  stmt.Embedding.Slice(),
		}
	}
	return result
}

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

	claims, ok := auth.GetUserFromContext(r.Context())
	if !ok || project.UserID.String() != claims.UserID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}

	// Check if we have embeddings client configured
	if s.embeddingClient == nil {
		respondError(w, http.StatusServiceUnavailable, "embedding service not configured - set OPENROUTER_API_KEY")
		return
	}

	// Analysis happens synchronously for now (could be made async with job queue)
	respondJSON(w, http.StatusAccepted, AnalysisStatusResponse{
		ProjectID: projectID,
		Status:    "ready",
		Progress:  100,
	})
}

// handleGetClusters returns clustering results for a project
func (s *Server) handleGetClustersImpl(w http.ResponseWriter, r *http.Request) {
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

	// Get statements for project
	statements, err := s.statementRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch statements")
		return
	}

	if len(statements) == 0 {
		respondJSON(w, http.StatusOK, []ClusterResponse{})
		return
	}

	// Convert to models.Statement
	modelStatements := s.convertToModelStatements(statements)

	// Get k parameter (optional)
	k := 0
	if kStr := r.URL.Query().Get("k"); kStr != "" {
		if kVal, err := strconv.Atoi(kStr); err == nil && kVal > 0 {
			k = kVal
		}
	}

	// Run clustering
	var result *clustering.ClusterResult
	if k > 0 {
		result = s.clusteringService.ClusterStatements(modelStatements, k)
	} else {
		result = s.clusteringService.AutoCluster(modelStatements, 10)
	}

	// Convert to response
	response := make([]ClusterResponse, len(result.Clusters))
	for i, c := range result.Clusters {
		keywords := make([]string, len(c.Keywords))
		for j, kw := range c.Keywords {
			keywords[j] = kw.Word
		}
		response[i] = ClusterResponse{
			ID:       c.ID,
			Keywords: keywords,
			Size:     c.Size,
			Density:  c.Density,
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetSimilarPairs returns similar pairs for a project
func (s *Server) handleGetSimilarPairsImpl(w http.ResponseWriter, r *http.Request) {
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

	// Parse optional threshold parameter
	threshold := 0.75
	if t := r.URL.Query().Get("threshold"); t != "" {
		if parsed, err := strconv.ParseFloat(t, 64); err == nil && parsed > 0 && parsed <= 1 {
			threshold = parsed
		}
	}

	// Get statements for project
	statements, err := s.statementRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch statements")
		return
	}

	if len(statements) == 0 {
		respondJSON(w, http.StatusOK, []SimilarPairResponse{})
		return
	}

	// Convert to models.Statement
	modelStatements := s.convertToModelStatements(statements)

	// Find similar pairs
	pairs := s.similarityService.FindSimilarStatements(modelStatements, threshold)

	// Convert to response
	response := make([]SimilarPairResponse, len(pairs))
	for i, p := range pairs {
		response[i] = SimilarPairResponse{
			Statement1: p.Statement1,
			Statement2: p.Statement2,
			File1:      p.File1,
			File2:      p.File2,
			Similarity: p.Similarity,
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetAnomalies returns anomaly detection results for a project
func (s *Server) handleGetAnomaliesImpl(w http.ResponseWriter, r *http.Request) {
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

	// Get statements for project
	statements, err := s.statementRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch statements")
		return
	}

	if len(statements) == 0 {
		respondJSON(w, http.StatusOK, []AnomalyResponse{})
		return
	}

	// Convert to models.Statement
	modelStatements := s.convertToModelStatements(statements)

	// Detect anomalies
	anomalies := s.anomalyService.GetAnomalies(modelStatements)

	// Convert to response
	response := make([]AnomalyResponse, len(anomalies))
	for i, a := range anomalies {
		response[i] = AnomalyResponse{
			Text:  a.Text,
			File:  a.File,
			Line:  a.Line,
			Score: a.Score,
		}
	}

	respondJSON(w, http.StatusOK, response)
}

// handleGetContradictions returns contradiction detection results for a project
func (s *Server) handleGetContradictionsImpl(w http.ResponseWriter, r *http.Request) {
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

	// Check if contradiction service is configured
	if s.contradictionService == nil {
		respondError(w, http.StatusServiceUnavailable, "contradiction detection not configured - set ANTHROPIC_API_KEY")
		return
	}

	// Get statements for project
	statements, err := s.statementRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch statements")
		return
	}

	if len(statements) == 0 {
		respondJSON(w, http.StatusOK, []ContradictionResponse{})
		return
	}

	// Convert to models.Statement
	modelStatements := s.convertToModelStatements(statements)

	// First find similar pairs (contradiction candidates)
	pairs := s.similarityService.FindSimilarStatements(modelStatements, 0.5)

	// Convert to statement pairs for contradiction analysis
	statementPairs := make([]contradiction.StatementPair, len(pairs))
	for i, p := range pairs {
		statementPairs[i] = contradiction.StatementPair{
			Statement1:   p.Statement1,
			Statement2:   p.Statement2,
			Statement1ID: modelStatements[p.Index1].ID,
			Statement2ID: modelStatements[p.Index2].ID,
			File1:        p.File1,
			File2:        p.File2,
			Similarity:   p.Similarity,
		}
	}

	// Detect contradictions
	contradictions, err := s.contradictionService.DetectContradictions(r.Context(), statementPairs)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to detect contradictions")
		return
	}

	// Convert to response
	response := make([]ContradictionResponse, len(contradictions))
	for i, c := range contradictions {
		response[i] = ContradictionResponse{
			Statement1:  c.Statement1,
			Statement2:  c.Statement2,
			File1:       c.File1,
			File2:       c.File2,
			Type:        string(c.Type),
			Severity:    string(c.Severity),
			Explanation: c.Explanation,
			Confidence:  c.Confidence,
		}
	}

	respondJSON(w, http.StatusOK, response)
}
