package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/todmy/doc-analyzer/internal/storage"
)

type Server struct {
	router        *chi.Mux
	db            *sql.DB
	projectRepo   storage.ProjectRepository
	documentRepo  storage.DocumentRepository
	statementRepo storage.StatementRepository
}

type ServerConfig struct {
	DB *sql.DB
}

func NewServer(config ServerConfig) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:*", "https://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	s := &Server{
		router:        r,
		db:            config.DB,
		projectRepo:   storage.NewPostgresProjectRepository(config.DB),
		documentRepo:  storage.NewPostgresDocumentRepository(config.DB),
		statementRepo: storage.NewPostgresStatementRepository(config.DB),
	}
	s.setupRoutes()

	return s
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.Get("/health", s.handleHealth)

	// API v1
	s.router.Route("/api/v1", func(r chi.Router) {
		// Auth routes (public)
		r.Post("/auth/register", s.handleRegister)
		r.Post("/auth/login", s.handleLogin)

		// Protected routes
		r.Group(func(r chi.Router) {
			// TODO: Add auth middleware
			// r.Use(s.authMiddleware)

			// Projects
			r.Route("/projects", func(r chi.Router) {
				r.Get("/", s.handleListProjectsImpl)
				r.Post("/", s.handleCreateProjectImpl)
				r.Get("/{projectID}", s.handleGetProjectImpl)
				r.Delete("/{projectID}", s.handleDeleteProjectImpl)

				// Documents
				r.Post("/{projectID}/documents", s.handleUpload)
				r.Get("/{projectID}/documents", s.handleListDocuments)
				r.Delete("/{projectID}/documents/{documentID}", s.handleDeleteDocument)

				// Analysis
				r.Post("/{projectID}/analyze", s.handleAnalyzeImpl)
				r.Get("/{projectID}/visualization", s.handleGetVisualizationImpl)
				r.Post("/{projectID}/visualization/axes", s.handleSetAxesImpl)

				// Results
				r.Get("/{projectID}/clusters", s.handleGetClustersImpl)
				r.Get("/{projectID}/similar-pairs", s.handleGetSimilarPairsImpl)
				r.Get("/{projectID}/anomalies", s.handleGetAnomaliesImpl)
				r.Get("/{projectID}/contradictions", s.handleGetContradictionsImpl)
			})
		})
	})

	// Serve static files for frontend
	s.router.Handle("/*", http.FileServer(http.Dir("web/dist")))
}

func (s *Server) Run(addr string) error {
	return http.ListenAndServe(addr, s.router)
}

// Helper to send JSON responses
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}
