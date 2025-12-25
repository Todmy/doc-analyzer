package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/todmy/doc-analyzer/internal/storage"
)

// ProjectRequest represents a project creation request
type ProjectRequest struct {
	Name string `json:"name"`
}

// ProjectResponse represents a project in API responses
type ProjectResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// handleListProjects returns all projects for the authenticated user
func (s *Server) handleListProjectsImpl(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	projects, err := s.projectRepo.GetByUserID(r.Context(), uid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch projects")
		return
	}

	response := make([]ProjectResponse, 0, len(projects))
	for _, p := range projects {
		response = append(response, ProjectResponse{
			ID:        p.ID.String(),
			Name:      p.Name,
			CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}

	respondJSON(w, http.StatusOK, response)
}

// handleCreateProject creates a new project
func (s *Server) handleCreateProjectImpl(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req ProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "name is required")
		return
	}

	project := &storage.Project{
		UserID: uid,
		Name:   req.Name,
	}

	if err := s.projectRepo.Create(r.Context(), project); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create project")
		return
	}

	respondJSON(w, http.StatusCreated, ProjectResponse{
		ID:        project.ID.String(),
		Name:      project.Name,
		CreatedAt: project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleGetProject returns a specific project
func (s *Server) handleGetProjectImpl(w http.ResponseWriter, r *http.Request) {
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

	project, err := s.projectRepo.GetByID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch project")
		return
	}

	if project == nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	// Verify ownership
	userID := getUserIDFromContext(r.Context())
	if project.UserID.String() != userID {
		respondError(w, http.StatusForbidden, "access denied")
		return
	}

	respondJSON(w, http.StatusOK, ProjectResponse{
		ID:        project.ID.String(),
		Name:      project.Name,
		CreatedAt: project.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt: project.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// handleDeleteProject deletes a project
func (s *Server) handleDeleteProjectImpl(w http.ResponseWriter, r *http.Request) {
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

	// Verify ownership
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

	if err := s.projectRepo.Delete(r.Context(), pid); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete project")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// getUserIDFromContext extracts user ID from request context
func getUserIDFromContext(ctx context.Context) string {
	// This would be set by auth middleware
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}
