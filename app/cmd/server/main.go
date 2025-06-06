package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
	"github.com/robfig/cron/v3"
)

var (
	Version   string = "dev"
	Commit    string = "unknown"
	BuildTime string = "unknown"
)

func main() {
	var (
		showHelp    = flag.Bool("help", false, "Show help message")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *showHelp {
		fmt.Printf("Article Summarizer v3 Server\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nEnvironment Variables:\n")
		fmt.Printf("  GEMINI_API_KEY        Gemini API key (required)\n")
		fmt.Printf("  SLACK_BOT_TOKEN       Slack bot token (required)\n")
		fmt.Printf("  PORT                  Server port (default: 8080)\n")
		fmt.Printf("  HOST                  Server host (default: 0.0.0.0)\n")
		fmt.Printf("  RSS_FEEDS             RSS feed configurations\n")
		fmt.Printf("  CACHE_TYPE            Cache type: memory or cloud-storage (default: memory)\n")
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("Article Summarizer v3 Server\n")
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Build Time: %s\n", BuildTime)
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create server
	server, err := handlers.NewServer(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Setup routes
	router := server.SetupRoutes()

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Setup individual RSS feed processing with cron scheduler (v1 style)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create cron scheduler
	c := cron.New()

	// Schedule each RSS feed individually with different timing
	for feedName, feedConfig := range cfg.RSSFeeds {
		if !feedConfig.Enabled {
			log.Printf("Skipping disabled feed: %s", feedName)
			continue
		}

		feedName := feedName // capture for closure
		_, err := c.AddFunc(feedConfig.Schedule, func() {
			log.Printf("🕐 Scheduled execution starting for %s", feedName)
			if err := server.ProcessSingleFeed(ctx, feedName); err != nil {
				log.Printf("❌ Scheduled processing failed for %s: %v", feedName, err)
			} else {
				log.Printf("✅ Scheduled processing completed for %s", feedName)
			}
		})

		if err != nil {
			log.Printf("❌ Failed to schedule feed %s: %v", feedName, err)
		} else {
			log.Printf("📅 Scheduled feed %s with cron: %s", feedConfig.Name, feedConfig.Schedule)
		}
	}

	// Start cron scheduler
	c.Start()
	defer c.Stop()

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server
	go func() {
		log.Printf("🚀 Starting server on %s:%s", cfg.Host, cfg.Port)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for shutdown signal
	<-sigChan
	log.Println("🛑 Shutting down server...")

	// Cancel background tasks
	cancel()

	// Stop cron scheduler
	c.Stop()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("✅ Server stopped")
}
