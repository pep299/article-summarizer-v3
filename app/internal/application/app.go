package application

import (
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handler"
	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/service"
)

// Application represents the application with all business logic components
type Application struct {
	Config           *config.Config
	ProcessHandler   *handler.Process
	WebhookHandler   *handler.Webhook
	cleanup          func() error
}

// New creates a new application instance with all dependencies
func New() (*Application, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Initialize cache manager
	cacheManager, err := cache.NewCloudStorageCache()
	if err != nil {
		return nil, fmt.Errorf("creating cache manager: %w", err)
	}

	// Create repositories (now with direct implementations)
	rssRepo := repository.NewRSSRepository()
	geminiRepo := repository.NewGeminiRepository(cfg.GeminiAPIKey, cfg.GeminiModel)
	cacheRepo := repository.NewCacheRepository(cacheManager)
	slackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.SlackChannel)
	webhookSlackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.WebhookSlackChannel)

	// Create services (business logic)
	feedService := service.NewFeed(rssRepo, cacheRepo, geminiRepo, slackRepo)
	urlService := service.NewURL(geminiRepo, webhookSlackRepo)

	// Create handlers (HTTP layer)
	processHandler := handler.NewProcess(feedService, cfg)
	webhookHandler := handler.NewWebhook(urlService)

	// Cleanup function
	cleanup := func() error {
		if cacheManager != nil {
			return cacheManager.Close()
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