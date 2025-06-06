package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
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

	// Setup routes and delegate to existing handler
	router := server.SetupRoutes()
	router.ServeHTTP(w, r)
}

// CloudEventData represents the data structure for Cloud Scheduler events
type CloudEventData struct {
	FeedName string `json:"feedName"`
}

// ProcessRSSScheduled processes RSS feeds triggered by Cloud Scheduler
func ProcessRSSScheduled(ctx context.Context, e CloudEvent) error {
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
	if err := json.Unmarshal(e.Data, &data); err != nil {
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
		for feedName, feedConfig := range cfg.RSSFeeds {
			if feedConfig.Enabled {
				if err := server.ProcessSingleFeed(ctx, feedName); err != nil {
					log.Printf("❌ Failed to process feed %s: %v", feedName, err)
				} else {
					log.Printf("✅ Successfully processed feed: %s", feedName)
				}
			}
		}
	}

	return nil
}

// CloudEvent represents a Cloud Function event
type CloudEvent struct {
	ID              string                 `json:"id"`
	Source          string                 `json:"source"`
	SpecVersion     string                 `json:"specversion"`
	Type            string                 `json:"type"`
	Subject         string                 `json:"subject"`
	Time            time.Time              `json:"time"`
	DataContentType string                 `json:"datacontenttype"`
	Data            json.RawMessage        `json:"data"`
	Extensions      map[string]interface{} `json:"extensions"`
}

func main() {
	// This main function is required for Cloud Functions
	// The actual function registration happens in init()
}
