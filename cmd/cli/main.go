package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
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
		fmt.Printf("Article Summarizer v3 CLI\n\n")
		fmt.Printf("Usage: %s [options]\n\n", os.Args[0])
		fmt.Printf("Description:\n")
		fmt.Printf("  Processes RSS feeds and generates article summaries using Gemini AI.\n")
		fmt.Printf("  Sends summaries to Slack channels.\n\n")
		fmt.Printf("Options:\n")
		flag.PrintDefaults()
		fmt.Printf("\nEnvironment Variables:\n")
		fmt.Printf("  GEMINI_API_KEY        Gemini API key (required)\n")
		fmt.Printf("  SLACK_BOT_TOKEN       Slack bot token (required)\n")
		fmt.Printf("  RSS_FEEDS             RSS feed configurations\n")
		fmt.Printf("  CACHE_TYPE            Cache type: memory or cloud-storage (default: memory)\n")
		os.Exit(0)
	}

	if *showVersion {
		fmt.Printf("Article Summarizer v3 CLI\n")
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
