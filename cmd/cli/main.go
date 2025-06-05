package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func main() {
	var (
		configFile = flag.String("config", "", "Configuration file path")
		command    = flag.String("cmd", "process", "Command to run: process, test-rss, test-gemini, test-slack")
		url        = flag.String("url", "", "URL to summarize (for test-gemini)")
		title      = flag.String("title", "", "Title for summarization")
		message    = flag.String("message", "", "Message to send to Slack (for test-slack)")
	)
	flag.Parse()

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

	switch *command {
	case "process":
		err = server.ProcessAndNotify(ctx)
		if err != nil {
			log.Fatalf("Processing failed: %v", err)
		}
		fmt.Println("Processing completed successfully")

	case "test-rss":
		err = testRSS(server)
		if err != nil {
			log.Fatalf("RSS test failed: %v", err)
		}

	case "test-gemini":
		if *url == "" {
			log.Fatal("URL is required for Gemini test")
		}
		err = testGemini(server, *url, *title)
		if err != nil {
			log.Fatalf("Gemini test failed: %v", err)
		}

	case "test-slack":
		if *message == "" {
			log.Fatal("Message is required for Slack test")
		}
		err = testSlack(server, *message)
		if err != nil {
			log.Fatalf("Slack test failed: %v", err)
		}

	default:
		fmt.Printf("Unknown command: %s\n", *command)
		flag.Usage()
		os.Exit(1)
	}
}

func testRSS(server *handlers.Server) error {
	fmt.Println("Testing RSS feeds...")
	
	// This is a bit of a hack to access the RSS client through reflection
	// In a real implementation, you'd want to expose these methods properly
	ctx := context.Background()
	
	// For now, just trigger the processing which will test RSS feeds
	return server.ProcessAndNotify(ctx)
}

func testGemini(server *handlers.Server, url, title string) error {
	fmt.Printf("Testing Gemini API with URL: %s\n", url)
	
	// Create a mock RSS item to test summarization
	// In a real implementation, you'd want to expose the Gemini client
	ctx := context.Background()
	
	// For now, just print that we would test Gemini
	fmt.Printf("Would summarize: %s - %s\n", title, url)
	return nil
}

func testSlack(server *handlers.Server, message string) error {
	fmt.Printf("Testing Slack notification with message: %s\n", message)
	
	// For now, just print that we would test Slack
	fmt.Printf("Would send to Slack: %s\n", message)
	return nil
}
