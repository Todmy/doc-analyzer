package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

type Server struct {
	router *chi.Mux
}

func NewServer() *Server {
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

	s := &Server{router: r}
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
				r.Get("/", s.handleListProjects)
				r.Post("/", s.handleCreateProject)
				r.Get("/{projectID}", s.handleGetProject)
				r.Delete("/{projectID}", s.handleDeleteProject)

				// Analysis
				r.Post("/{projectID}/analyze", s.handleAnalyze)
				r.Get("/{projectID}/visualization", s.handleGetVisualization)
				r.Post("/{projectID}/visualization/axes", s.handleSetAxes)

				// Results
				r.Get("/{projectID}/clusters", s.handleGetClusters)
				r.Get("/{projectID}/similar-pairs", s.handleGetSimilarPairs)
				r.Get("/{projectID}/anomalies", s.handleGetAnomalies)
				r.Get("/{projectID}/contradictions", s.handleGetContradictions)
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
