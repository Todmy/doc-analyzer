package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/todmy/doc-analyzer/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/doc_analyzer?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	openRouterKey := os.Getenv("OPENROUTER_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	server := api.NewServer(api.ServerConfig{
		DB:              db,
		JWTSecret:       jwtSecret,
		OpenRouterKey:   openRouterKey,
		AnthropicAPIKey: anthropicKey,
	})

	fmt.Printf("Starting doc-analyzer server on port %s\n", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
