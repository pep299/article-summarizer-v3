package main

import (
	"log"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/server"
)

func main() {
	// Create handler
	handler, cleanup, err := server.CreateHandler()
	if err != nil {
		log.Fatalf("Failed to create handler: %v", err)
	}
	defer cleanup()

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}