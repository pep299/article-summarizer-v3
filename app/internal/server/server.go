package server

import (
	"log"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/di"
	"github.com/pep299/article-summarizer-v3/internal/handler"
	"github.com/pep299/article-summarizer-v3/internal/middleware"
	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/service"
)

// CreateHandler creates the main HTTP handler for the application
func CreateHandler() (http.Handler, func(), error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, err
	}

	// Create DI container
	container, err := di.NewContainer(cfg)
	if err != nil {
		return nil, nil, err
	}

	// Create repositories
	rssRepo := repository.NewRSSRepository(container.RSSClient)
	geminiRepo := repository.NewGeminiRepository(container.GeminiClient)
	cacheRepo := repository.NewCacheRepository(container.CacheManager)
	slackRepo := repository.NewSlackRepository(container.SlackClient)
	webhookSlackRepo := repository.NewSlackRepository(container.WebhookSlackClient)

	// Create services
	feedService := service.NewFeed(rssRepo, cacheRepo, geminiRepo, slackRepo)
	urlService := service.NewURL(geminiRepo, webhookSlackRepo)

	// Create handlers
	processHandler := handler.NewProcess(feedService, cfg)
	webhookHandler := handler.NewWebhook(urlService)

	// Create auth middleware
	authMiddleware := middleware.Auth(cfg.WebhookAuthToken)

	// Setup routes
	mux := http.NewServeMux()
	mux.Handle("/process", authMiddleware(processHandler))
	mux.Handle("/webhook", authMiddleware(webhookHandler))
	mux.Handle("/", authMiddleware(processHandler)) // Default to process

	// Return handler and cleanup function
	cleanup := func() {
		container.Close()
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