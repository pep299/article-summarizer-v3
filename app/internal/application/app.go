package application

import (
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/handler"
	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/service"
	"github.com/pep299/article-summarizer-v3/internal/slack"
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

	// Initialize low-level clients
	cacheManager, err := cache.NewCloudStorageCache()
	if err != nil {
		return nil, fmt.Errorf("creating cache manager: %w", err)
	}

	rssClient := rss.NewClient()
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	slackClient := slack.NewClient(cfg.SlackBotToken, cfg.SlackChannel)
	webhookSlackClient := slack.NewClient(cfg.SlackBotToken, cfg.WebhookSlackChannel)

	// Create repositories (wrappers around clients)
	rssRepo := repository.NewRSSRepository(rssClient)
	geminiRepo := repository.NewGeminiRepository(geminiClient)
	cacheRepo := repository.NewCacheRepository(cacheManager)
	slackRepo := repository.NewSlackRepository(slackClient)
	webhookSlackRepo := repository.NewSlackRepository(webhookSlackClient)

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