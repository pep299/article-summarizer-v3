package service

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/service/feed"
	"github.com/pep299/article-summarizer-v3/internal/service/limiter"
)

type Feed struct {
	rss       repository.RSSRepository
	processed repository.ProcessedArticleRepository
	gemini    repository.GeminiRepository
	slack     repository.SlackRepository
	registry  *feed.FeedRegistry
	limiter   limiter.ArticleLimiter
}

func NewFeed(
	rss repository.RSSRepository,
	processed repository.ProcessedArticleRepository,
	gemini repository.GeminiRepository,
	slack repository.SlackRepository,
	limiter limiter.ArticleLimiter,
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
		limiter:   limiter,
	}
}

func (f *Feed) Process(ctx context.Context, feedName string) error {
	strategy, exists := f.registry.GetStrategy(feedName)
	if !exists {
		return fmt.Errorf("unknown feed: %s", feedName)
	}

	processedIndex, err := f.processed.LoadIndex(ctx)
	if err != nil {
		log.Printf("Error loading processed articles index for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return fmt.Errorf("loading processed articles index: %w", err)
	}

	articles, err := f.fetchAndPrepareArticles(ctx, strategy, feedName)
	if err != nil {
		log.Printf("Error fetching articles for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return err
	}

	if len(articles) == 0 {
		log.Printf("📋 新着記事はありませんでした: %s", strategy.GetConfig().DisplayName)
		return nil
	}

	unprocessedArticles := f.selectUnprocessedArticles(articles, processedIndex, strategy.GetConfig().DisplayName)
	if len(unprocessedArticles) == 0 {
		log.Printf("✅ 新着記事はありません: %s", strategy.GetConfig().DisplayName)
		return nil
	}

	if err := f.processArticles(ctx, unprocessedArticles, strategy.GetConfig().DisplayName); err != nil {
		log.Printf("Error processing articles for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return err
	}
	return nil
}

// fetchAndPrepareArticles handles RSS fetching, parsing, and basic preparation
func (f *Feed) fetchAndPrepareArticles(ctx context.Context, strategy feed.FeedStrategy, feedName string) ([]repository.Item, error) {
	config := strategy.GetConfig()
	log.Printf("📡 RSS取得開始: %s", config.DisplayName)

	// Fetch RSS XML using strategy-specific headers
	headers := strategy.GetRequestHeaders()
	xmlContent, err := f.rss.FetchFeedXML(ctx, config.URL, headers)
	if err != nil {
		log.Printf("Error fetching RSS XML for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return nil, fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	// Parse XML using strategy-specific parser
	articles, err := strategy.ParseFeed(xmlContent)
	if err != nil {
		log.Printf("Error parsing RSS feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return nil, fmt.Errorf("parsing RSS feed %s: %w", feedName, err)
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

	log.Printf("📄 %d件の記事を取得: %s", len(articles), config.DisplayName)

	// Apply feed-specific filtering and deduplication
	uniqueArticles := f.rss.GetUniqueItems(articles)
	filteredArticles := strategy.FilterItems(uniqueArticles)

	log.Printf("After filtering: %d articles remain from %s", len(filteredArticles), config.DisplayName)
	return filteredArticles, nil
}

// selectUnprocessedArticles filters out already processed articles and applies test limits
func (f *Feed) selectUnprocessedArticles(articles []repository.Item, processedIndex map[string]*repository.IndexEntry, displayName string) []repository.Item {
	// Check against processed articles using in-memory index
	var unprocessedArticles []repository.Item
	for _, article := range articles {
		key := f.processed.GenerateKey(article)
		if !f.processed.IsProcessed(key, processedIndex) {
			unprocessedArticles = append(unprocessedArticles, article)
		}
	}

	// Apply article limiting
	unprocessedArticles = f.limiter.Limit(unprocessedArticles)

	log.Printf("Processing %d new articles from %s", len(unprocessedArticles), displayName)
	return unprocessedArticles
}

// processArticles handles the summarization and notification for all articles
func (f *Feed) processArticles(ctx context.Context, articles []repository.Item, displayName string) error {
	for _, article := range articles {
		if err := f.processArticle(ctx, article); err != nil {
			log.Printf("Error processing article %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
	}

	log.Printf("🎉 %s処理完了: %d件成功", displayName, len(articles))
	return nil
}

func (f *Feed) processArticle(ctx context.Context, article repository.Item) error {
	log.Printf("🔍 記事処理開始: %s", article.Title)

	summary, err := f.gemini.SummarizeURL(ctx, article.Link)
	if err != nil {
		log.Printf("Error summarizing article %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("summarizing article: %w", err)
	}

	articleSummary := repository.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	if err := f.slack.SendArticleSummary(ctx, articleSummary); err != nil {
		log.Printf("Error sending Slack notification for article %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("sending Slack notification: %w", err)
	}

	// Mark as processed (includes GCS re-fetch and update)
	if err := f.processed.MarkAsProcessed(ctx, article); err != nil {
		log.Printf("Error marking article as processed %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("marking as processed: %w", err)
	}

	log.Printf("✅ 記事処理完了: %s", article.Title)
	return nil
}
