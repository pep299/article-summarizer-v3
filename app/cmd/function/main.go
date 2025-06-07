package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func init() {
	// Register HTTP function for manual triggers and webhooks
	functions.HTTP("SummarizeArticles", SummarizeArticles)
}

// SummarizeArticles is the HTTP function for webhook requests and manual triggers
func SummarizeArticles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Failed to load configuration: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Create server instance
	server, err := handlers.NewServer(cfg)
	if err != nil {
		log.Printf("Failed to create server: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Simple path-based routing
	switch r.URL.Path {
	case "/", "/process":
		// RSS feed processing (scheduled)
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check Bearer token authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if cfg.WebhookAuthToken != "" && token != cfg.WebhookAuthToken {
			http.Error(w, "Invalid token", http.StatusForbidden)
			return
		}

		// Parse JSON payload
		var payload struct {
			FeedName string `json:"feedName"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		if payload.FeedName == "" {
			http.Error(w, "Missing 'feedName' in payload", http.StatusBadRequest)
			return
		}

		log.Printf("🕐 Processing RSS feed via HTTP: %s", payload.FeedName)
		if err := server.ProcessSingleFeed(ctx, payload.FeedName); err != nil {
			log.Printf("❌ Failed to process feed %s: %v", payload.FeedName, err)
			http.Error(w, fmt.Sprintf("Failed to process feed: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Successfully processed RSS feed via HTTP: %s", payload.FeedName)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"feed":    payload.FeedName,
			"message": "Feed processed successfully",
		})

	case "/webhook":
		// Individual URL summarization (webhook)
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check Bearer token authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Missing Authorization header", http.StatusUnauthorized)
			return
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid Authorization header format", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		if cfg.WebhookAuthToken != "" && token != cfg.WebhookAuthToken {
			http.Error(w, "Invalid token", http.StatusForbidden)
			return
		}

		// Parse JSON payload
		var payload struct {
			URL string `json:"url"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}

		if payload.URL == "" {
			http.Error(w, "Missing 'url' in payload", http.StatusBadRequest)
			return
		}

		log.Printf("🔗 Processing individual URL: %s", payload.URL)
		if err := server.ProcessSingleURL(ctx, payload.URL); err != nil {
			log.Printf("❌ Failed to process URL %s: %v", payload.URL, err)
			http.Error(w, fmt.Sprintf("Failed to process URL: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Successfully processed URL: %s", payload.URL)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"url":     payload.URL,
			"message": "URL processed successfully",
		})

	default:
		http.NotFound(w, r)
	}
}

func main() {
	// This main function is required for Cloud Functions
	// The actual function registration happens in init()
}
