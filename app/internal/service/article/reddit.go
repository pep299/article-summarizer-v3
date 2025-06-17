package article

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/repository/rss"
	"github.com/pep299/article-summarizer-v3/internal/service/limiter"
)

type RedditProcessor struct {
	redditRepo    rss.FeedRepository
	geminiRepo    repository.GeminiRepository
	slackRepo     repository.SlackRepository
	processedRepo repository.ProcessedArticleRepository
	limiter       limiter.ArticleLimiter
}

func NewRedditProcessor(
	rssRepo repository.RSSRepository,
	geminiRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
	processedRepo repository.ProcessedArticleRepository,
	limiter limiter.ArticleLimiter,
) *RedditProcessor {
	return &RedditProcessor{
		redditRepo:    rss.NewRedditRSSRepository(rssRepo),
		geminiRepo:    geminiRepo,
		slackRepo:     slackRepo,
		processedRepo: processedRepo,
		limiter:       limiter,
	}
}

func (p *RedditProcessor) Process(ctx context.Context) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	logger.Printf("Process request started feed=reddit")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logger.Printf("Process request completed feed=reddit duration_ms=%d", duration.Milliseconds())
	}()

	// 1. データ取得
	logger.Printf("Feed processing started feed=reddit")
	articles, err := p.redditRepo.FetchArticles(ctx)
	if err != nil {
		logger.Printf("Error processing feed reddit: %v", err)
		return fmt.Errorf("processing feed reddit: %w", err)
	}

	// Filter unprocessed articles
	unprocessedArticles, err := filterUnprocessedArticles(ctx, p.processedRepo, articles)
	if err != nil {
		return fmt.Errorf("filtering unprocessed articles: %w", err)
	}

	// Apply article limiting
	limitedArticles := p.limiter.Limit(unprocessedArticles)

	logger.Printf("Selected unprocessed articles: %d from Reddit r/programming", len(limitedArticles))

	// Process each article
	for i, article := range limitedArticles {
		if err := p.processRedditArticle(ctx, article); err != nil {
			logger.Printf("Error processing article %s: %v", article.Title, err)
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
		logger.Printf("Article processed %d/%d title=%s", i+1, len(limitedArticles), article.Title)
	}

	logger.Printf("Feed processing completed feed=reddit processed_count=%d", len(limitedArticles))
	return nil
}

// processRedditArticle handles Reddit articles with comments
func (p *RedditProcessor) processRedditArticle(ctx context.Context, article repository.Item) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Article processing started title=%s url=%s comment_url=%s source=%s",
		article.Title, article.Link, article.CommentURL, article.Source)

	// 2. 記事要約
	summaryStart := time.Now()
	summary, err := p.geminiRepo.SummarizeURL(ctx, article.Link)
	if err != nil {
		logger.Printf("Error summarizing article %s: %v", article.Title, err)
		return fmt.Errorf("summarizing article: %w", err)
	}
	summaryDuration := time.Since(summaryStart)

	var slackDuration time.Duration

	// 3. コメント取得・要約 (Cloud Functions で動作しないため無効化)
	// if article.CommentURL != "" && article.CommentURL != article.Link {
	//     commentStart := time.Now()
	//     comments, err := p.redditRepo.FetchComments(ctx, article.CommentURL)
	//     if err != nil {
	//         logger.Printf("Error fetching comments for %s: %v", article.Title, err)
	//         return fmt.Errorf("fetching comments: %w", err)
	//     }
	//
	//     // 4. コメント要約
	//     commentSummary, err := p.geminiRepo.SummarizeComments(ctx, comments.Text)
	//     if err != nil {
	//         logger.Printf("Error summarizing comments for %s: %v", article.Title, err)
	//         return fmt.Errorf("summarizing comments: %w", err)
	//     }
	//     commentDuration = time.Since(commentStart)
	//
	//     // 5. 通知送信（記事 + コメント）
	//     // ... comment processing code disabled ...
	// }

	// 4. 通知送信（記事のみ）
	slackStart := time.Now()
	if err := p.slackRepo.Send(ctx, repository.Notification{
		Title:   article.Title,
		Source:  article.Source,
		URL:     article.Link,
		Summary: summary.Summary,
	}); err != nil {
		logger.Printf("Error sending notification for %s: %v", article.Title, err)
		return fmt.Errorf("sending notification: %w", err)
	}
	slackDuration = time.Since(slackStart)

	// 6. インデックス更新
	processStart := time.Now()
	if err := p.processedRepo.MarkAsProcessed(ctx, article); err != nil {
		logger.Printf("Error marking article as processed %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("marking as processed: %w", err)
	}
	processDuration := time.Since(processStart)

	totalDuration := time.Since(start)
	logger.Printf("Article processing completed title=%s total_duration_ms=%d summary_duration_ms=%d slack_duration_ms=%d process_duration_ms=%d (comment processing disabled)",
		article.Title, totalDuration.Milliseconds(), summaryDuration.Milliseconds(), slackDuration.Milliseconds(), processDuration.Milliseconds())

	return nil
}
