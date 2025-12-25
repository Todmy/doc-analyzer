package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/todmy/doc-analyzer/internal/anomaly"
	"github.com/todmy/doc-analyzer/internal/auth"
	"github.com/todmy/doc-analyzer/internal/clustering"
	"github.com/todmy/doc-analyzer/internal/contradiction"
	"github.com/todmy/doc-analyzer/internal/embeddings"
	"github.com/todmy/doc-analyzer/internal/similarity"
	"github.com/todmy/doc-analyzer/internal/storage"
	"github.com/todmy/doc-analyzer/internal/visualization"
)

type Server struct {
	router        *chi.Mux
	db            *sql.DB
	authService   auth.Service
	projectRepo   storage.ProjectRepository
	documentRepo  storage.DocumentRepository
	statementRepo storage.StatementRepository

	// Analysis services
	embeddingClient      *embeddings.Client
	clusteringService    *clustering.Service
	similarityService    *similarity.Service
	anomalyService       *anomaly.Service
	contradictionService *contradiction.Service
	visualizationService *visualization.Service
}

type ServerConfig struct {
	DB              *sql.DB
	JWTSecret       string
	OpenRouterKey   string
	AnthropicAPIKey string
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

	// Initialize auth service
	userRepo := auth.NewPostgresRepository(config.DB)
	jwtSecret := config.JWTSecret
	if jwtSecret == "" {
		jwtSecret = "development-secret-change-in-prod"
	}
	authService := auth.NewJWTService(auth.Config{
		SecretKey: jwtSecret,
	}, userRepo)

	// Initialize embedding client (optional - can work without it)
	var embClient *embeddings.Client
	if config.OpenRouterKey != "" {
		embClient = embeddings.NewClient(config.OpenRouterKey)
	}

	// Initialize analysis services
	clusteringSvc := clustering.NewService(clustering.DefaultConfig())
	similaritySvc := similarity.NewService(0.75)
	anomalySvc := anomaly.NewService(anomaly.DefaultConfig())

	// Initialize contradiction service (optional - needs API key)
	var contradictionSvc *contradiction.Service
	if config.AnthropicAPIKey != "" {
		analyzer := contradiction.NewAnalyzer(contradiction.Config{
			APIKey: config.AnthropicAPIKey,
		})
		contradictionSvc = contradiction.NewService(analyzer, contradiction.DefaultServiceConfig())
	}

	// Initialize visualization service
	visualizationSvc := visualization.NewService(visualization.DefaultConfig(), embClient)

	s := &Server{
		router:        r,
		db:            config.DB,
		authService:   authService,
		projectRepo:   storage.NewPostgresProjectRepository(config.DB),
		documentRepo:  storage.NewPostgresDocumentRepository(config.DB),
		statementRepo: storage.NewPostgresStatementRepository(config.DB),

		embeddingClient:      embClient,
		clusteringService:    clusteringSvc,
		similarityService:    similaritySvc,
		anomalyService:       anomalySvc,
		contradictionService: contradictionSvc,
		visualizationService: visualizationSvc,
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
			r.Use(auth.Middleware(s.authService))

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

	// Serve static files for frontend (SPA)
	s.router.Get("/*", s.serveSPA)
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

const staticDir = "web/dist"

// serveSPA serves the SPA - static files if they exist, otherwise index.html
func (s *Server) serveSPA(w http.ResponseWriter, r *http.Request) {
	// Get the requested path
	path := r.URL.Path

	// Clean the path
	path = filepath.Clean(path)
	if path == "/" {
		path = "/index.html"
	}

	// Build the full file path
	fullPath := filepath.Join(staticDir, path)

	// Check if file exists
	info, err := os.Stat(fullPath)
	if err == nil && !info.IsDir() {
		// File exists, serve it
		http.ServeFile(w, r, fullPath)
		return
	}

	// For directories or non-existent files, check if it's an asset request
	if strings.HasPrefix(path, "/assets/") {
		// Asset not found
		http.NotFound(w, r)
		return
	}

	// For SPA routes, serve index.html
	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); errors.Is(err, fs.ErrNotExist) {
		http.Error(w, "Frontend not built. Run 'make frontend-build' first.", http.StatusNotFound)
		return
	}

	http.ServeFile(w, r, indexPath)
}
