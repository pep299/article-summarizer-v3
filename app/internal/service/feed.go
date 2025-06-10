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

	processedIndex, err := f.processed.LoadIndex(ctx)
	if err != nil {
		return fmt.Errorf("loading processed articles index: %w", err)
	}

	articles, err := f.fetchAndPrepareArticles(ctx, strategy, feedName)
	if err != nil {
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

	return f.processArticles(ctx, unprocessedArticles, strategy.GetConfig().DisplayName)
}

// fetchAndPrepareArticles handles RSS fetching, parsing, and basic preparation
func (f *Feed) fetchAndPrepareArticles(ctx context.Context, strategy feed.FeedStrategy, feedName string) ([]repository.Item, error) {
	config := strategy.GetConfig()
	log.Printf("📡 RSS取得開始: %s", config.DisplayName)

	// Fetch RSS XML using strategy-specific headers
	headers := strategy.GetRequestHeaders()
	xmlContent, err := f.rss.FetchFeedXML(ctx, config.URL, headers)
	if err != nil {
		return nil, fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	// Parse XML using strategy-specific parser
	articles, err := strategy.ParseFeed(xmlContent)
	if err != nil {
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

	// Apply test environment article limit
	if testLimit := os.Getenv("TEST_MAX_ARTICLES"); testLimit != "" {
		if limit, err := strconv.Atoi(testLimit); err == nil && limit > 0 && limit < len(unprocessedArticles) {
			log.Printf("TEST_MAX_ARTICLES制限により %d件に制限", limit)
			unprocessedArticles = unprocessedArticles[:limit]
		}
	}

	log.Printf("Processing %d new articles from %s", len(unprocessedArticles), displayName)
	return unprocessedArticles
}

// processArticles handles the summarization and notification for all articles
func (f *Feed) processArticles(ctx context.Context, articles []repository.Item, displayName string) error {
	for _, article := range articles {
		if err := f.processArticle(ctx, article); err != nil {
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
