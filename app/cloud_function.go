package main

import (
	"context"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func init() {
	functions.HTTP("SummarizeArticles", SummarizeArticles)
}

func SummarizeArticles(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Use the existing handlers.Server for Cloud Functions
	// TODO: Migrate to the new server architecture when ready
	server, err := handlers.NewServer(cfg)
	if err != nil {
		log.Printf("Failed to create server: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer server.Close()

	// Use the existing routing logic from cmd/function/main.go
	switch r.URL.Path {
	case "/", "/process":
		handleProcess(w, r, server, cfg)
	case "/webhook":
		handleWebhook(w, r, server, cfg)
	default:
		http.NotFound(w, r)
	}
}

func handleProcess(w http.ResponseWriter, r *http.Request, server *handlers.Server, cfg *config.Config) {
	// Implementation would be moved from cmd/function/main.go
	// For now, keep the existing implementation
}

func handleWebhook(w http.ResponseWriter, r *http.Request, server *handlers.Server, cfg *config.Config) {
	// Implementation would be moved from cmd/function/main.go
	// For now, keep the existing implementation
}

func main() {
	// Required for Cloud Functions
}