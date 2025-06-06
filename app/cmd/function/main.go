package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func init() {
	// Register HTTP function for manual triggers and webhooks
	functions.HTTP("SummarizeArticles", SummarizeArticles)

	// Register Cloud Scheduler function for scheduled RSS processing
	functions.CloudEvent("ProcessRSSScheduled", ProcessRSSScheduled)
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
	case "/process":
		// Check authentication for POST requests
		if r.Method == "POST" {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Failed to parse form data", http.StatusBadRequest)
				return
			}

			token := r.FormValue("token")
			if cfg.WebhookAuthToken != "" && token != cfg.WebhookAuthToken {
				http.Error(w, "Unauthorized", http.StatusForbidden)
				return
			}
		}

		// Get feed name from query parameter
		feedName := r.URL.Query().Get("feed")
		if feedName == "" {
			http.Error(w, "Missing 'feed' query parameter", http.StatusBadRequest)
			return
		}

		// Process the specific RSS feed
		log.Printf("🕐 Processing RSS feed via HTTP: %s", feedName)
		if err := server.ProcessSingleFeed(ctx, feedName); err != nil {
			log.Printf("❌ Failed to process feed %s: %v", feedName, err)
			http.Error(w, fmt.Sprintf("Failed to process feed: %v", err), http.StatusInternalServerError)
			return
		}

		log.Printf("✅ Successfully processed RSS feed via HTTP: %s", feedName)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "success",
			"feed":    feedName,
			"message": "Feed processed successfully",
		})

	default:
		http.NotFound(w, r)
	}
}

// CloudEventData represents the data structure for Cloud Scheduler events
type CloudEventData struct {
	FeedName string `json:"feedName"`
}

// ProcessRSSScheduled processes RSS feeds triggered by Cloud Scheduler
func ProcessRSSScheduled(ctx context.Context, e event.Event) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Create server instance
	server, err := handlers.NewServer(cfg)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Parse the Cloud Event data
	var data CloudEventData
	if err := json.Unmarshal(e.Data(), &data); err != nil {
		return fmt.Errorf("failed to parse event data: %w", err)
	}

	// Process the specific RSS feed
	if data.FeedName != "" {
		log.Printf("🕐 Processing scheduled RSS feed: %s", data.FeedName)
		if err := server.ProcessSingleFeed(ctx, data.FeedName); err != nil {
			return fmt.Errorf("failed to process RSS feed %s: %w", data.FeedName, err)
		}
		log.Printf("✅ Successfully processed RSS feed: %s", data.FeedName)
	} else {
		// Process all enabled feeds if no specific feed is specified
		log.Printf("🕐 Processing all enabled RSS feeds")
		feeds := []string{"hatena", "lobsters"}
		for _, feedName := range feeds {
			if err := server.ProcessSingleFeed(ctx, feedName); err != nil {
				log.Printf("❌ Failed to process feed %s: %v", feedName, err)
			} else {
				log.Printf("✅ Successfully processed feed: %s", feedName)
			}
		}
	}

	return nil
}

func main() {
	// This main function is required for Cloud Functions
	// The actual function registration happens in init()
}
