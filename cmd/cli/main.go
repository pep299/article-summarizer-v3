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

	// Process RSS feeds and send notifications
	err = server.ProcessAndNotify(ctx)
	if err != nil {
		log.Fatalf("Processing failed: %v", err)
	}
	fmt.Println("Processing completed successfully")
}
