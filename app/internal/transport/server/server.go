package server

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/application"
	"github.com/pep299/article-summarizer-v3/internal/transport/middleware"
)

// CreateHandler creates the main HTTP handler for the application
func CreateHandler() (http.Handler, func(), error) {
	// Create application (handles all DI and business logic)
	app, err := application.New()
	if err != nil {
		log.Printf("Error creating application: %v", err)
		return nil, nil, err
	}

	// Create auth middleware
	authMiddleware := middleware.Auth(app.Config.WebhookAuthToken)

	// Setup routes (pure HTTP routing)
	mux := http.NewServeMux()
	mux.Handle("POST /process", authMiddleware(app.ProcessHandler))
	mux.Handle("POST /webhook", authMiddleware(app.WebhookHandler))
	mux.Handle("GET /x", authMiddleware(app.XHandler))                       // X fetch endpoint (auth required)
	mux.Handle("GET /x/quote-chain", authMiddleware(app.XQuoteChainHandler)) // X quote chain endpoint (auth required)
	mux.HandleFunc("GET /hc", healthCheck)                                   // Health check endpoint

	// Return handler and cleanup function
	cleanup := func() {
		app.Close()
	}

	return mux, cleanup, nil
}

// HandleRequest handles a single HTTP request (for Cloud Functions)
func HandleRequest(w http.ResponseWriter, r *http.Request) {
	handler, cleanup, err := CreateHandler()
	if err != nil {
		log.Printf("Failed to create handler: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer cleanup()

	handler.ServeHTTP(w, r)
}

// healthCheck handles health check requests
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{
		"status": "ok",
	}
	json.NewEncoder(w).Encode(response)
}
