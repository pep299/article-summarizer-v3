package server

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/pep299/article-summarizer-v3/internal/application"
	"github.com/pep299/article-summarizer-v3/internal/transport/middleware"
)

// CreateHandler creates the main HTTP handler for the application
func CreateHandler() (http.Handler, func(), error) {
	// Create application (handles all DI and business logic)
	app, err := application.New()
	if err != nil {
		log.Printf("Error creating application: %v\nStack:\n%s", err, debug.Stack())
		return nil, nil, err
	}

	// Create auth middleware
	authMiddleware := middleware.Auth(app.Config.WebhookAuthToken)

	// Setup routes (pure HTTP routing)
	mux := http.NewServeMux()
	mux.Handle("/process", authMiddleware(app.ProcessHandler))
	mux.Handle("/webhook", authMiddleware(app.WebhookHandler))
	mux.Handle("/", authMiddleware(app.ProcessHandler)) // Default to process

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
		log.Printf("Failed to create handler: %v\nStack:\n%s", err, debug.Stack())
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	defer cleanup()

	handler.ServeHTTP(w, r)
}
