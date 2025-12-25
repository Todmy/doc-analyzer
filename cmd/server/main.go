package main

import (
	"fmt"
	"log"
	"os"

	"github.com/todmy/doc-analyzer/internal/api"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := api.NewServer()

	fmt.Printf("Starting doc-analyzer server on port %s\n", port)
	if err := server.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
