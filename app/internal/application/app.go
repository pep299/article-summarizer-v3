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
	Config             *Config
	WebhookHandler     *handler.Webhook
	XHandler           *handler.X
	XQuoteChainHandler *handler.XQuoteChain
	HatenaHandler      *handler.HatenaHandler
	RedditHandler      *handler.RedditHandler
	LobstersHandler    *handler.LobstersHandler
	cleanup            func() error
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
	geminiRepo := repository.NewGeminiRepository(cfg.GeminiAPIKey, cfg.GeminiModel, cfg.GeminiBaseURL)
	processedRepo, err := repository.NewProcessedArticleRepository()
	if err != nil {
		return nil, fmt.Errorf("creating processed article repository: %w", err)
	}
	redditSlackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.SlackChannelReddit, cfg.SlackBaseURL)
	hatenaSlackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.SlackChannelHatena, cfg.SlackBaseURL)
	lobstersSlackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.SlackChannelLobsters, cfg.SlackBaseURL)
	webhookSlackRepo := repository.NewSlackRepository(cfg.SlackBotToken, cfg.WebhookSlackChannel, cfg.SlackBaseURL)

	// Create services (business logic) - use production limiter by default
	articleLimiter := limiter.NewProductionArticleLimiter()
	urlService := service.NewURL(geminiRepo, webhookSlackRepo)

	// Create X repository
	xRepo := repository.NewXClient()

	// Create handlers (HTTP layer)
	webhookHandler := handler.NewWebhook(urlService)
	xHandler := handler.NewX(xRepo)
	xQuoteChainHandler := handler.NewXQuoteChain(xRepo)
	hatenaHandler := handler.NewHatenaHandler(rssRepo, geminiRepo, hatenaSlackRepo, processedRepo, articleLimiter)
	redditHandler := handler.NewRedditHandler(rssRepo, geminiRepo, redditSlackRepo, processedRepo, articleLimiter)
	lobstersHandler := handler.NewLobstersHandler(rssRepo, geminiRepo, lobstersSlackRepo, processedRepo, articleLimiter)

	// Cleanup function
	cleanup := func() error {
		if processedRepo != nil {
			return processedRepo.Close()
		}
		return nil
	}

	return &Application{
		Config:             cfg,
		WebhookHandler:     webhookHandler,
		XHandler:           xHandler,
		XQuoteChainHandler: xQuoteChainHandler,
		HatenaHandler:      hatenaHandler,
		RedditHandler:      redditHandler,
		LobstersHandler:    lobstersHandler,
		cleanup:            cleanup,
	}, nil
}

// Close cleans up application resources
func (a *Application) Close() error {
	if a.cleanup != nil {
		return a.cleanup()
	}
	return nil
}
