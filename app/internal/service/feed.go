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
		logger.Printf("Error fetching articles for feed %s: %v", feedName, err)
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
		logger.Printf("Error processing articles for feed %s: %v", feedName, err)
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
		logger.Printf("Error fetching RSS XML for feed %s: %v", feedName, err)
		return nil, fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	fetchDuration := time.Since(start)
	logger.Printf("RSS XML fetch completed feed=%s xml_length=%d duration_ms=%d", feedName, len(xmlContent), fetchDuration.Milliseconds())

	// Parse XML using strategy-specific parser
	parseStart := time.Now()
	articles, err := strategy.ParseFeed(xmlContent)
	if err != nil {
		logger.Printf("Error parsing RSS feed %s: %v", feedName, err)
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
		if err := f.ProcessArticle(ctx, article); err != nil {
			logger.Printf("Error processing article %s: %v", article.Title, err)
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

func (f *Feed) ProcessArticle(ctx context.Context, article repository.Item) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Article processing started title=%s url=%s comment_url=%s source=%s", 
		article.Title, article.Link, article.CommentURL, article.Source)

	// Phase 1: Summarize article content
	summaryStart := time.Now()
	summary, err := f.gemini.SummarizeURL(ctx, article.Link)
	if err != nil {
		logger.Printf("Error summarizing article %s: %v", article.Title, err)
		return fmt.Errorf("summarizing article: %w", err)
	}
	summaryDuration := time.Since(summaryStart)

	// Phase 2: Send article summary notification
	slackStart := time.Now()
	if err := f.slack.SendArticleSummary(ctx, repository.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}); err != nil {
		logger.Printf("Error sending article Slack notification for %s: %v", article.Title, err)
		return fmt.Errorf("sending article Slack notification: %w", err)
	}
	slackDuration1 := time.Since(slackStart)

	// Phase 3: Generate and send comment summary (if has comments)
	var commentDuration, slackDuration2 time.Duration
	if article.CommentURL != "" && article.CommentURL != article.Link {
		commentStart := time.Now()
		commentSummary, err := f.generateCommentSummary(ctx, article)
		if err != nil {
			logger.Printf("Error generating comment summary for %s: %v", article.Title, err)
			return fmt.Errorf("generating comment summary: %w", err)
		}
		commentDuration = time.Since(commentStart)

		// Send comment summary notification
		slackStart2 := time.Now()
		if err := f.slack.SendCommentSummary(ctx, article, commentSummary); err != nil {
			logger.Printf("Error sending comment Slack notification for %s: %v", article.Title, err)
			return fmt.Errorf("sending comment Slack notification: %w", err)
		}
		slackDuration2 = time.Since(slackStart2)
	}

	// Phase 4: Mark as processed
	processStart := time.Now()
	if err := f.processed.MarkAsProcessed(ctx, article); err != nil {
		logger.Printf("Error marking article as processed %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("marking as processed: %w", err)
	}
	processDuration := time.Since(processStart)

	totalDuration := time.Since(start)
	if article.CommentURL != "" && article.CommentURL != article.Link {
		logger.Printf("Article processing completed title=%s total_duration_ms=%d summary_duration_ms=%d comment_duration_ms=%d slack1_duration_ms=%d slack2_duration_ms=%d process_duration_ms=%d",
			article.Title, totalDuration.Milliseconds(), summaryDuration.Milliseconds(), commentDuration.Milliseconds(), slackDuration1.Milliseconds(), slackDuration2.Milliseconds(), processDuration.Milliseconds())
	} else {
		logger.Printf("Article processing completed title=%s total_duration_ms=%d summary_duration_ms=%d slack_duration_ms=%d process_duration_ms=%d",
			article.Title, totalDuration.Milliseconds(), summaryDuration.Milliseconds(), slackDuration1.Milliseconds(), processDuration.Milliseconds())
	}
	return nil
}

// generateCommentSummary generates a summary of comments/discussions for any source
func (f *Feed) generateCommentSummary(ctx context.Context, article repository.Item) (string, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	
	logger.Printf("Generating comment summary for %s from %s", article.Source, article.CommentURL)
	
	switch article.Source {
	case "reddit":
		return f.generateRedditCommentSummary(ctx, article.CommentURL)
	case "hatena", "lobsters":
		// 将来的にHatenaやLobstersのコメント要約も対応可能
		return "", fmt.Errorf("comment summarization not yet implemented for %s", article.Source)
	default:
		return "", fmt.Errorf("unknown source for comment summarization: %s", article.Source)
	}
}

// generateRedditCommentSummary generates a summary of Reddit comments for a given post URL
func (f *Feed) generateRedditCommentSummary(ctx context.Context, redditURL string) (string, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	
	// Get Reddit strategy from registry
	strategy, exists := f.registry.GetStrategy("reddit")
	if !exists {
		return "", fmt.Errorf("reddit strategy not found")
	}
	
	redditStrategy, ok := strategy.(*feed.RedditStrategy)
	if !ok {
		return "", fmt.Errorf("strategy is not a Reddit strategy")
	}
	
	// Fetch comments from Reddit using JSON API
	logger.Printf("Fetching Reddit comments from %s using JSON API", redditURL)
	comments, err := redditStrategy.FetchComments(ctx, redditURL, f.rss)
	if err != nil {
		return "", fmt.Errorf("fetching Reddit comments: %w", err)
	}
	
	if len(comments) == 0 {
		return "", fmt.Errorf("no comments found for Reddit post")
	}
	
	logger.Printf("Successfully fetched %d Reddit comments from JSON API", len(comments))
	
	// Combine comments into text
	commentsText := redditStrategy.CombineCommentsText(comments)
	if commentsText == "" {
		return "", fmt.Errorf("no comment text to summarize")
	}
	
	logger.Printf("Combined comments text length: %d characters", len(commentsText))
	
	// Summarize the comments using specialized comment prompt
	summary, err := f.gemini.SummarizeComments(ctx, commentsText)
	if err != nil {
		return "", fmt.Errorf("summarizing Reddit comments: %w", err)
	}
	
	logger.Printf("Successfully generated Reddit comment summary")
	return summary, nil
}
