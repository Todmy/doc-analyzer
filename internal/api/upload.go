package api

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/todmy/doc-analyzer/internal/auth"
	"github.com/todmy/doc-analyzer/internal/storage"
)

const maxUploadSize = 10 << 20 // 10 MB

// UploadResponse represents the response after file upload
type UploadResponse struct {
	DocumentID string `json:"document_id"`
	Filename   string `json:"filename"`
	Hash       string `json:"hash"`
	Status     string `json:"status"`
}

// handleUpload handles document file uploads
func (s *Server) handleUpload(w http.ResponseWriter, r *http.Request) {
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

	// Limit upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)

	// Parse multipart form
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		respondError(w, http.StatusBadRequest, "file too large or invalid form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, http.StatusBadRequest, "no file provided")
		return
	}
	defer file.Close()

	// Validate file extension
	ext := filepath.Ext(header.Filename)
	allowedExts := map[string]bool{".md": true, ".txt": true, ".json": true, ".csv": true}
	if !allowedExts[ext] {
		respondError(w, http.StatusBadRequest, "only .md, .txt, .json, and .csv files are allowed")
		return
	}

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	// Calculate content hash
	hash := sha256.Sum256(content)
	hashStr := hex.EncodeToString(hash[:])

	// Check if document with same hash already exists
	existingDoc, err := s.documentRepo.GetByHash(r.Context(), pid, hashStr)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to check existing documents")
		return
	}

	if existingDoc != nil {
		respondJSON(w, http.StatusOK, UploadResponse{
			DocumentID: existingDoc.ID.String(),
			Filename:   existingDoc.Filename,
			Hash:       hashStr,
			Status:     "exists",
		})
		return
	}

	// Create new document
	doc := &storage.Document{
		ProjectID:   pid,
		Filename:    header.Filename,
		Content:     string(content),
		ContentHash: hashStr,
	}

	if err := s.documentRepo.Create(r.Context(), doc); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save document")
		return
	}

	// Extract statements from document
	statements := extractStatements(doc.Content, doc.ID, ext)

	if len(statements) > 0 {
		// Generate embeddings for statements
		if err := s.generateEmbeddingsForStatements(r.Context(), statements); err != nil {
			// Log error but don't fail the upload
			// Statements will be stored without embeddings
		}

		// Save statements
		if err := s.statementRepo.CreateBatch(r.Context(), statements); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to save statements")
			return
		}
	}

	respondJSON(w, http.StatusCreated, UploadResponse{
		DocumentID: doc.ID.String(),
		Filename:   doc.Filename,
		Hash:       hashStr,
		Status:     "created",
	})
}

// handleListDocuments lists all documents in a project
func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
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

	docs, err := s.documentRepo.GetByProjectID(r.Context(), pid)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch documents")
		return
	}

	type DocumentResponse struct {
		ID       string `json:"id"`
		Filename string `json:"filename"`
		Hash     string `json:"hash"`
	}

	response := make([]DocumentResponse, 0, len(docs))
	for _, doc := range docs {
		response = append(response, DocumentResponse{
			ID:       doc.ID.String(),
			Filename: doc.Filename,
			Hash:     doc.ContentHash,
		})
	}

	respondJSON(w, http.StatusOK, response)
}

// handleDeleteDocument deletes a document from a project
func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	projectID := chi.URLParam(r, "projectID")
	documentID := chi.URLParam(r, "documentID")

	if projectID == "" || documentID == "" {
		respondError(w, http.StatusBadRequest, "project id and document id are required")
		return
	}

	did, err := uuid.Parse(documentID)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid document id")
		return
	}

	// Delete statements first, then document
	if err := s.statementRepo.DeleteByDocumentID(r.Context(), did); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete statements")
		return
	}

	if err := s.documentRepo.Delete(r.Context(), did); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to delete document")
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
