package handlers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

// Interfaces for dependency injection and testing
type RSSClient interface {
	FetchFeed(ctx context.Context, feedName, url string) ([]rss.Item, error)
}

type GeminiClient interface {
	SummarizeURL(ctx context.Context, url string) (*gemini.SummarizeResponse, error)
}

type SlackClient interface {
	SendArticleSummary(ctx context.Context, summary slack.ArticleSummary) error
}

// Server holds the dependencies for RSS processing
type Server struct {
	config             *config.Config
	rssClient          RSSClient
	geminiClient       GeminiClient
	slackClient        SlackClient
	webhookSlackClient SlackClient
	cacheManager       *cache.CloudStorageCache
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize cache manager
	cacheManager, err := cache.NewCloudStorageCache()
	if err != nil {
		return nil, fmt.Errorf("creating cache manager: %w", err)
	}

	return &Server{
		config:             cfg,
		rssClient:          rss.NewClient(),
		geminiClient:       gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel),
		slackClient:        slack.NewClient(cfg.SlackBotToken, cfg.SlackChannel),
		webhookSlackClient: slack.NewClient(cfg.SlackBotToken, cfg.WebhookSlackChannel),
		cacheManager:       cacheManager,
	}, nil
}

// NewServerWithDeps creates a new server instance with provided dependencies (for testing)
func NewServerWithDeps(cfg *config.Config, rssClient RSSClient, geminiClient GeminiClient, slackClient SlackClient, webhookSlackClient SlackClient, cacheManager *cache.CloudStorageCache) *Server {
	return &Server{
		config:             cfg,
		rssClient:          rssClient,
		geminiClient:       geminiClient,
		slackClient:        slackClient,
		webhookSlackClient: slackClient, // Use same client for testing
		cacheManager:       cacheManager,
	}
}

// ProcessSingleFeed processes a single RSS feed (v1 style: sync processing)
func (s *Server) ProcessSingleFeed(ctx context.Context, feedName string) error {
	var feedDisplayName, feedURL string

	switch feedName {
	case "hatena":
		feedDisplayName = "はてブ テクノロジー"
		feedURL = s.config.HatenaRSSURL
	case "lobsters":
		feedDisplayName = "Lobsters"
		feedURL = s.config.LobstersRSSURL
	default:
		return fmt.Errorf("feed %s not found", feedName)
	}

	log.Printf("📡 RSS取得開始: %s", feedDisplayName)

	// 1. RSS記事取得 (sync, no concurrency)
	articles, err := s.rssClient.FetchFeed(ctx, feedDisplayName, feedURL)
	if err != nil {
		return fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	log.Printf("📄 %d件の記事を取得: %s", len(articles), feedDisplayName)

	if len(articles) == 0 {
		log.Printf("📋 新着記事はありませんでした: %s", feedDisplayName)
		return nil
	}

	// 2. 重複除去
	uniqueArticles := rss.GetUniqueItems(articles)
	log.Printf("Found %d unique articles from %s", len(uniqueArticles), feedDisplayName)

	// 3. フィルタリング (only remove "ask" category)
	filteredArticles := rss.FilterItems(uniqueArticles)
	log.Printf("After filtering: %d articles remain from %s", len(filteredArticles), feedDisplayName)

	// 4. キャッシュチェック（既に処理済みを除外）
	uncachedArticles := []rss.Item{}
	for _, article := range filteredArticles {
		if s.cacheManager != nil {
			cached, err := cache.IsCached(ctx, s.cacheManager, article)
			if err != nil {
				return fmt.Errorf("checking cache for article %s: %w", article.Title, err)
			}
			if !cached {
				uncachedArticles = append(uncachedArticles, article)
			}
		} else {
			// If cache manager is nil, treat all articles as uncached
			uncachedArticles = append(uncachedArticles, article)
		}
	}

	log.Printf("Processing %d new articles from %s", len(uncachedArticles), feedDisplayName)

	if len(uncachedArticles) == 0 {
		log.Printf("✅ 新着記事はありません: %s", feedDisplayName)
		return nil
	}

	// 5. 各記事を同期処理（v1 style: 1つずつ処理）
	for _, article := range uncachedArticles {
		if err := s.processArticle(ctx, article); err != nil {
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
	}

	log.Printf("🎉 %s処理完了: %d件成功", feedDisplayName, len(uncachedArticles))
	return nil
}

// ProcessSingleURL processes a single URL and sends summary to webhook Slack channel
func (s *Server) ProcessSingleURL(ctx context.Context, url string) error {
	startTime := time.Now()
	log.Printf("🔍 オンデマンド記事処理開始: %s", url)

	// Check if Gemini client is available
	if s.geminiClient == nil {
		return fmt.Errorf("gemini client is not initialized")
	}

	// Gemini APIで要約
	summary, err := s.geminiClient.SummarizeURL(ctx, url)
	if err != nil {
		return fmt.Errorf("summarizing URL: %w", err)
	}

	// Check if webhook Slack client is available
	if s.webhookSlackClient == nil {
		return fmt.Errorf("webhook slack client is not initialized")
	}

	// Create a minimal RSS item for the URL
	article := rss.Item{
		Title:  url, // Use URL as title since SummarizeResponse doesn't have Title
		Link:   url,
		Source: "オンデマンドリクエスト",
	}

	// Send to webhook Slack channel using the webhook client
	articleSummary := slack.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	if err := s.webhookSlackClient.SendArticleSummary(ctx, articleSummary); err != nil {
		return fmt.Errorf("sending webhook Slack notification: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("✅ オンデマンド記事処理完了: %s (所要時間: %v)", url, duration)
	return nil
}

// processArticle processes a single article (sync, like v1)
func (s *Server) processArticle(ctx context.Context, article rss.Item) error {
	startTime := time.Now()
	log.Printf("🔍 記事処理開始: %s", article.Title)

	// Check if Gemini client is available
	if s.geminiClient == nil {
		return fmt.Errorf("gemini client is not initialized")
	}

	// Gemini APIで要約 (sync)
	summary, err := s.geminiClient.SummarizeURL(ctx, article.Link)
	if err != nil {
		return fmt.Errorf("summarizing article: %w", err)
	}

	// Check if Slack client is available
	if s.slackClient == nil {
		return fmt.Errorf("slack client is not initialized")
	}

	// Slack通知 (1件ずつ)
	articleSummary := slack.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	if err := s.slackClient.SendArticleSummary(ctx, articleSummary); err != nil {
		return fmt.Errorf("sending Slack notification: %w", err)
	}

	// Slack通知成功後にキャッシュに保存
	if s.cacheManager != nil {
		if err := cache.MarkAsProcessed(ctx, s.cacheManager, article); err != nil {
			return fmt.Errorf("caching article: %w", err)
		}
	}

	duration := time.Since(startTime)
	log.Printf("✅ 記事処理完了: %s (所要時間: %v)", article.Title, duration)
	return nil
}

// Close closes the server and cleans up resources
func (s *Server) Close() error {
	if s.cacheManager != nil {
		return s.cacheManager.Close()
	}
	return nil
}
