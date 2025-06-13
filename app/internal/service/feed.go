package service

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
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
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Feed processing started feed=%s", feedName)

	strategy, exists := f.registry.GetStrategy(feedName)
	if !exists {
		logger.Printf("Unknown feed requested feed=%s", feedName)
		return fmt.Errorf("unknown feed: %s", feedName)
	}

	processedIndex, err := f.processed.LoadIndex(ctx)
	if err != nil {
		logger.Printf("Error loading processed articles index for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return fmt.Errorf("loading processed articles index: %w", err)
	}

	articles, err := f.fetchAndPrepareArticles(ctx, strategy, feedName)
	if err != nil {
		logger.Printf("Error fetching articles for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return err
	}

	if len(articles) == 0 {
		logger.Printf("No new articles found feed=%s display_name=%s", feedName, strategy.GetConfig().DisplayName)
		return nil
	}

	unprocessedArticles := f.selectUnprocessedArticles(articles, processedIndex, strategy.GetConfig().DisplayName)
	if len(unprocessedArticles) == 0 {
		logger.Printf("No unprocessed articles feed=%s display_name=%s", feedName, strategy.GetConfig().DisplayName)
		return nil
	}

	if err := f.processArticles(ctx, unprocessedArticles, strategy.GetConfig().DisplayName); err != nil {
		logger.Printf("Error processing articles for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return err
	}

	totalDuration := time.Since(start)
	logger.Printf("Feed processing completed feed=%s processed_count=%d total_duration_ms=%d",
		feedName, len(unprocessedArticles), totalDuration.Milliseconds())
	return nil
}

// fetchAndPrepareArticles handles RSS fetching, parsing, and basic preparation
func (f *Feed) fetchAndPrepareArticles(ctx context.Context, strategy feed.FeedStrategy, feedName string) ([]repository.Item, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	config := strategy.GetConfig()
	start := time.Now()

	logger.Printf("RSS fetch started feed=%s display_name=%s url=%s", feedName, config.DisplayName, config.URL)

	// Fetch RSS XML using strategy-specific headers
	headers := strategy.GetRequestHeaders()
	xmlContent, err := f.rss.FetchFeedXML(ctx, config.URL, headers)
	if err != nil {
		logger.Printf("Error fetching RSS XML for feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return nil, fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	fetchDuration := time.Since(start)
	logger.Printf("RSS XML fetch completed feed=%s xml_length=%d duration_ms=%d", feedName, len(xmlContent), fetchDuration.Milliseconds())

	// Parse XML using strategy-specific parser
	parseStart := time.Now()
	articles, err := strategy.ParseFeed(xmlContent)
	if err != nil {
		logger.Printf("Error parsing RSS feed %s: %v\nStack:\n%s", feedName, err, debug.Stack())
		return nil, fmt.Errorf("parsing RSS feed %s: %w", feedName, err)
	}

	parseDuration := time.Since(parseStart)
	logger.Printf("RSS parse completed feed=%s articles_count=%d parse_duration_ms=%d", feedName, len(articles), parseDuration.Milliseconds())

	// Set source and parse dates for all items
	for i := range articles {
		articles[i].Source = feedName
		if articles[i].PubDate != "" {
			if parsedDate, err := strategy.ParseDate(articles[i].PubDate); err == nil {
				articles[i].ParsedDate = parsedDate
			}
		}
	}

	// Apply feed-specific filtering and deduplication
	uniqueArticles := f.rss.GetUniqueItems(articles)
	filteredArticles := strategy.FilterItems(uniqueArticles)

	totalDuration := time.Since(start)
	logger.Printf("RSS processing completed feed=%s original_count=%d filtered_count=%d total_duration_ms=%d",
		feedName, len(articles), len(filteredArticles), totalDuration.Milliseconds())
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

	originalCount := len(unprocessedArticles)
	// Apply article limiting
	unprocessedArticles = f.limiter.Limit(unprocessedArticles)

	if originalCount != len(unprocessedArticles) {
		log.Printf("Article limiting applied: %d -> %d articles from %s", originalCount, len(unprocessedArticles), displayName)
	}
	log.Printf("Selected unprocessed articles: %d from %s", len(unprocessedArticles), displayName)
	return unprocessedArticles
}

// processArticles handles the summarization and notification for all articles
func (f *Feed) processArticles(ctx context.Context, articles []repository.Item, displayName string) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Article batch processing started display_name=%s count=%d", displayName, len(articles))

	for i, article := range articles {
		articleStart := time.Now()
		if err := f.processArticle(ctx, article); err != nil {
			logger.Printf("Error processing article %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
		articleDuration := time.Since(articleStart)
		logger.Printf("Article processed %d/%d title=%s duration_ms=%d", i+1, len(articles), article.Title, articleDuration.Milliseconds())
	}

	totalDuration := time.Since(start)
	logger.Printf("Article batch processing completed display_name=%s success_count=%d total_duration_ms=%d avg_duration_ms=%d",
		displayName, len(articles), totalDuration.Milliseconds(), totalDuration.Milliseconds()/int64(len(articles)))
	return nil
}

func (f *Feed) processArticle(ctx context.Context, article repository.Item) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Article processing started title=%s url=%s", article.Title, article.Link)

	// Summarization phase
	summaryStart := time.Now()
	summary, err := f.gemini.SummarizeURL(ctx, article.Link)
	if err != nil {
		logger.Printf("Error summarizing article %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("summarizing article: %w", err)
	}
	summaryDuration := time.Since(summaryStart)

	articleSummary := repository.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	// Slack notification phase
	slackStart := time.Now()
	if err := f.slack.SendArticleSummary(ctx, articleSummary); err != nil {
		logger.Printf("Error sending Slack notification for article %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("sending Slack notification: %w", err)
	}
	slackDuration := time.Since(slackStart)

	// Mark as processed phase
	processStart := time.Now()
	if err := f.processed.MarkAsProcessed(ctx, article); err != nil {
		logger.Printf("Error marking article as processed %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("marking as processed: %w", err)
	}
	processDuration := time.Since(processStart)

	totalDuration := time.Since(start)
	logger.Printf("Article processing completed title=%s total_duration_ms=%d summary_duration_ms=%d slack_duration_ms=%d process_duration_ms=%d",
		article.Title, totalDuration.Milliseconds(), summaryDuration.Milliseconds(), slackDuration.Milliseconds(), processDuration.Milliseconds())
	return nil
}
