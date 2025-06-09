package di

import (
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

// Container holds all dependencies
type Container struct {
	Config             *config.Config
	RSSClient          *rss.Client
	GeminiClient       *gemini.Client
	SlackClient        *slack.Client
	WebhookSlackClient *slack.Client
	CacheManager       *cache.CloudStorageCache
}

// NewContainer creates a new dependency container
func NewContainer(cfg *config.Config) (*Container, error) {
	// Initialize cache manager
	cacheManager, err := cache.NewCloudStorageCache()
	if err != nil {
		return nil, fmt.Errorf("creating cache manager: %w", err)
	}

	// Initialize clients using existing implementations
	rssClient := rss.NewClient()
	geminiClient := gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel)
	slackClient := slack.NewClient(cfg.SlackBotToken, cfg.SlackChannel)
	webhookSlackClient := slack.NewClient(cfg.SlackBotToken, cfg.WebhookSlackChannel)

	return &Container{
		Config:             cfg,
		RSSClient:          rssClient,
		GeminiClient:       geminiClient,
		SlackClient:        slackClient,
		WebhookSlackClient: webhookSlackClient,
		CacheManager:       cacheManager,
	}, nil
}

// Close cleans up resources
func (c *Container) Close() error {
	if c.CacheManager != nil {
		return c.CacheManager.Close()
	}
	return nil
}