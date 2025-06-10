package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/service/feed"
)

type Feed struct {
	rss       repository.RSSRepository
	processed repository.ProcessedArticleRepository
	gemini    repository.GeminiRepository
	slack     repository.SlackRepository
	registry  *feed.FeedRegistry
}

func NewFeed(
	rss repository.RSSRepository,
	processed repository.ProcessedArticleRepository,
	gemini repository.GeminiRepository,
	slack repository.SlackRepository,
) *Feed {
	// Initialize feed registry and register all default strategies
	registry := feed.NewFeedRegistry()

	// Register all default feeds - no hard-coded names here!
	for _, strategy := range feed.GetDefaultFeeds() {
		registry.Register(strategy)
	}

	return &Feed{
		rss:       rss,
		processed: processed,
		gemini:    gemini,
		slack:     slack,
		registry:  registry,
	}
}

func (f *Feed) Process(ctx context.Context, feedName string) error {
	strategy, exists := f.registry.GetStrategy(feedName)
	if !exists {
		return fmt.Errorf("unknown feed: %s", feedName)
	}

	feedConfig := strategy.GetConfig()
	feedURL := feedConfig.URL
	displayName := feedConfig.DisplayName
	log.Printf("📡 RSS取得開始: %s", displayName)

	// Cloud Function startup: Load processed articles index
	processedIndex, err := f.processed.LoadIndex(ctx)
	if err != nil {
		return fmt.Errorf("loading processed articles index: %w", err)
	}
	log.Printf("📋 処理済み記事インデックス読み込み完了: %d件", len(processedIndex))

	// Fetch RSS XML using strategy-specific headers
	headers := strategy.GetRequestHeaders()
	xmlContent, err := f.rss.FetchFeedXML(ctx, feedURL, headers)
	if err != nil {
		return fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	// Parse XML using strategy-specific parser
	articles, err := strategy.ParseFeed(xmlContent)
	if err != nil {
		return fmt.Errorf("parsing RSS feed %s: %w", feedName, err)
	}

	// Set source and parse dates for all items
	for i := range articles {
		articles[i].Source = feedName
		if articles[i].PubDate != "" {
			if parsedDate, err := strategy.ParseDate(articles[i].PubDate); err == nil {
				articles[i].ParsedDate = parsedDate
			}
		}
	}

	log.Printf("📄 %d件の記事を取得: %s", len(articles), displayName)

	if len(articles) == 0 {
		log.Printf("📋 新着記事はありませんでした: %s", displayName)
		return nil
	}

	// 重複除去とフィルタリング
	uniqueArticles := f.rss.GetUniqueItems(articles)
	filteredArticles := strategy.FilterItems(uniqueArticles)

	log.Printf("After filtering: %d articles remain from %s", len(filteredArticles), displayName)

	// Check against processed articles using in-memory index
	var unprocessedArticles []repository.Item
	for _, article := range filteredArticles {
		key := f.processed.GenerateKey(article)
		if !f.processed.IsProcessed(key, processedIndex) {
			unprocessedArticles = append(unprocessedArticles, article)
		}
	}

	// テスト環境での記事数制限
	if testLimit := os.Getenv("TEST_MAX_ARTICLES"); testLimit != "" {
		if limit, err := strconv.Atoi(testLimit); err == nil && limit > 0 && limit < len(unprocessedArticles) {
			log.Printf("TEST_MAX_ARTICLES制限により %d件に制限", limit)
			unprocessedArticles = unprocessedArticles[:limit]
		}
	}

	log.Printf("Processing %d new articles from %s", len(unprocessedArticles), displayName)

	if len(unprocessedArticles) == 0 {
		log.Printf("✅ 新着記事はありません: %s", displayName)
		return nil
	}

	// 各記事を処理
	for _, article := range unprocessedArticles {
		if err := f.processArticle(ctx, article); err != nil {
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
	}

	log.Printf("🎉 %s処理完了: %d件成功", displayName, len(unprocessedArticles))
	return nil
}

func (f *Feed) processArticle(ctx context.Context, article repository.Item) error {
	log.Printf("🔍 記事処理開始: %s", article.Title)

	summary, err := f.gemini.SummarizeURL(ctx, article.Link)
	if err != nil {
		return fmt.Errorf("summarizing article: %w", err)
	}

	articleSummary := repository.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	if err := f.slack.SendArticleSummary(ctx, articleSummary); err != nil {
		return fmt.Errorf("sending Slack notification: %w", err)
	}

	// Mark as processed (includes GCS re-fetch and update)
	if err := f.processed.MarkAsProcessed(ctx, article); err != nil {
		return fmt.Errorf("marking as processed: %w", err)
	}

	log.Printf("✅ 記事処理完了: %s", article.Title)
	return nil
}
