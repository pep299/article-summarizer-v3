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

// Server holds the dependencies for RSS processing
type Server struct {
	config       *config.Config
	rssClient    *rss.Client
	geminiClient *gemini.Client
	slackClient  *slack.Client
	cacheManager *cache.Manager
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize cache manager
	cacheManager, err := cache.NewManager(cfg.CacheType, time.Duration(cfg.CacheDuration)*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("creating cache manager: %w", err)
	}

	return &Server{
		config:       cfg,
		rssClient:    rss.NewClient(),
		geminiClient: gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel),
		slackClient:  slack.NewClient(cfg.SlackBotToken, cfg.SlackChannel),
		cacheManager: cacheManager,
	}, nil
}

// ProcessSingleFeed processes a single RSS feed (v1 style: sync processing)
func (s *Server) ProcessSingleFeed(ctx context.Context, feedName string) error {
	feedConfig, exists := s.config.RSSFeeds[feedName]
	if !exists {
		return fmt.Errorf("feed %s not found", feedName)
	}

	if !feedConfig.Enabled {
		log.Printf("Feed %s is disabled, skipping", feedName)
		return nil
	}

	log.Printf("📡 RSS取得開始: %s", feedConfig.Name)

	// 1. RSS記事取得 (sync, no concurrency)
	articles, err := s.rssClient.FetchFeed(ctx, feedConfig.Name, feedConfig.URL)
	if err != nil {
		return fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	log.Printf("📄 %d件の記事を取得: %s", len(articles), feedConfig.Name)

	if len(articles) == 0 {
		log.Printf("📋 新着記事はありませんでした: %s", feedConfig.Name)
		return nil
	}

	// 2. 重複除去
	uniqueArticles := rss.GetUniqueItems(articles)
	log.Printf("Found %d unique articles from %s", len(uniqueArticles), feedConfig.Name)

	// 3. フィルタリング (only remove "ask" category)
	filteredArticles := rss.FilterItems(uniqueArticles)
	log.Printf("After filtering: %d articles remain from %s", len(filteredArticles), feedConfig.Name)

	// 4. キャッシュチェック（既に処理済みを除外）
	uncachedArticles := []rss.Item{}
	for _, article := range filteredArticles {
		cached, err := s.cacheManager.IsCached(ctx, article)
		if err != nil {
			log.Printf("Error checking cache for %s: %v", article.Title, err)
			continue
		}
		if !cached {
			uncachedArticles = append(uncachedArticles, article)
		}
	}

	log.Printf("Processing %d new articles from %s", len(uncachedArticles), feedConfig.Name)

	if len(uncachedArticles) == 0 {
		log.Printf("✅ 新着記事はありません: %s", feedConfig.Name)
		return nil
	}

	// 5. 各記事を同期処理（v1 style: 1つずつ処理）
	successCount := 0
	for _, article := range uncachedArticles {
		if err := s.processArticle(ctx, article); err != nil {
			log.Printf("❌ 記事処理エラー (%s): %v", article.Title, err)
		} else {
			successCount++
		}
	}

	log.Printf("🎉 %s処理完了: %d/%d件成功", feedConfig.Name, successCount, len(uncachedArticles))
	return nil
}

// processArticle processes a single article (sync, like v1)
func (s *Server) processArticle(ctx context.Context, article rss.Item) error {
	startTime := time.Now()
	log.Printf("🔍 記事処理開始: %s", article.Title)

	// Gemini APIで要約 (sync)
	summary, err := s.geminiClient.SummarizeURL(ctx, article.Link)
	if err != nil {
		return fmt.Errorf("summarizing article: %w", err)
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
	if err := s.cacheManager.SetSummary(ctx, article, *summary); err != nil {
		log.Printf("Error caching summary for %s: %v", article.Title, err)
	}

	duration := time.Since(startTime)
	log.Printf("✅ 記事処理完了: %s (所要時間: %v)", article.Title, duration)
	return nil
}