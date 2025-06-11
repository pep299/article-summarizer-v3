package application

import (
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/service"
	"github.com/pep299/article-summarizer-v3/internal/service/limiter"
	"github.com/pep299/article-summarizer-v3/internal/transport/handler"
)

// Application represents the application with all business logic components
type Application struct {
	Config         *Config
	ProcessHandler *handler.Process
	WebhookHandler *handler.Webhook
	cleanup        func() error
}

// New creates a new application instance with all dependencies
func New() (*Application, error) {
	// Load configuration
	cfg, err := Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Create repositories (now with direct implementations)
	rssRepo := repository.NewRSSRepository()
	geminiRepo := repository.NewGeminiRepository(cfg.GeminiAPIKey, cfg.GeminiModel)
	processedRepo, err := repository.NewProcessedArticleRepository()
	if err != nil {
		return nil, fmt.Errorf("creating processed article repository: %w", err)
	}
	slackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.SlackChannel)
	webhookSlackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.WebhookSlackChannel)

	// Create services (business logic) - use production limiter by default
	articleLimiter := limiter.NewProductionArticleLimiter()
	feedService := service.NewFeed(rssRepo, processedRepo, geminiRepo, slackRepo, articleLimiter)
	urlService := service.NewURL(geminiRepo, webhookSlackRepo)

	// Create handlers (HTTP layer)
	processHandler := handler.NewProcess(feedService)
	webhookHandler := handler.NewWebhook(urlService)

	// Cleanup function
	cleanup := func() error {
		if processedRepo != nil {
			return processedRepo.Close()
		}
		return nil
	}

	return &Application{
		Config:         cfg,
		ProcessHandler: processHandler,
		WebhookHandler: webhookHandler,
		cleanup:        cleanup,
	}, nil
}

// Close cleans up application resources
func (a *Application) Close() error {
	if a.cleanup != nil {
		return a.cleanup()
	}
	return nil
}
