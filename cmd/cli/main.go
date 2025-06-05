package main

import (
	"context"
	"fmt"
	"log"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create server instance (contains all the clients)
	server, err := handlers.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	ctx := context.Background()

	// Process all enabled RSS feeds one by one (v1 style)
	for feedName, feedConfig := range cfg.RSSFeeds {
		if !feedConfig.Enabled {
			log.Printf("Skipping disabled feed: %s", feedName)
			continue
		}

		log.Printf("🚀 Processing feed: %s", feedConfig.Name)
		err = server.ProcessSingleFeed(ctx, feedName)
		if err != nil {
			log.Printf("❌ Processing failed for %s: %v", feedConfig.Name, err)
		} else {
			log.Printf("✅ Processing completed for %s", feedConfig.Name)
		}
	}

	fmt.Println("🎉 All feeds processing completed successfully")
}
