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

type HatenaProcessor struct {
	hatenaRepo    rss.FeedRepository
	hatenaRSSRepo *rss.HatenaRSSRepository
	geminiRepo    repository.GeminiRepository
	slackRepo     repository.SlackRepository
	processedRepo repository.ProcessedArticleRepository
	limiter       limiter.ArticleLimiter
}

func NewHatenaProcessor(
	rssRepo repository.RSSRepository,
	geminiRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
	processedRepo repository.ProcessedArticleRepository,
	limiter limiter.ArticleLimiter,
) *HatenaProcessor {
	hatenaRSSRepo := rss.NewHatenaRSSRepository(rssRepo)
	return &HatenaProcessor{
		hatenaRepo:    hatenaRSSRepo,
		hatenaRSSRepo: hatenaRSSRepo,
		geminiRepo:    geminiRepo,
		slackRepo:     slackRepo,
		processedRepo: processedRepo,
		limiter:       limiter,
	}
}

func (p *HatenaProcessor) Process(ctx context.Context) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	logger.Printf("Process request started feed=hatena")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		logger.Printf("Process request completed feed=hatena duration_ms=%d", duration.Milliseconds())
	}()

	// 1. データ取得
	logger.Printf("Feed processing started feed=hatena")
	articles, err := p.hatenaRepo.FetchArticles(ctx)
	if err != nil {
		logger.Printf("Error processing feed hatena: %v", err)
		return fmt.Errorf("processing feed hatena: %w", err)
	}

	// Filter unprocessed articles
	unprocessedArticles, err := filterUnprocessedArticles(ctx, p.processedRepo, articles)
	if err != nil {
		return fmt.Errorf("filtering unprocessed articles: %w", err)
	}

	// Apply article limiting
	limitedArticles := p.limiter.Limit(unprocessedArticles)

	logger.Printf("Selected unprocessed articles: %d from はてブ テクノロジー", len(limitedArticles))

	// Process each article
	for i, article := range limitedArticles {
		if err := p.processHatenaArticle(ctx, article); err != nil {
			logger.Printf("Error processing article %s: %v", article.Title, err)
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
		logger.Printf("Article processed %d/%d title=%s", i+1, len(limitedArticles), article.Title)
	}

	logger.Printf("Feed processing completed feed=hatena processed_count=%d", len(limitedArticles))
	return nil
}

// processHatenaArticle handles articles with Hatena bookmark comments
func (p *HatenaProcessor) processHatenaArticle(ctx context.Context, article repository.Item) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Article processing started title=%s url=%s source=%s",
		article.Title, article.Link, article.Source)

	// 2. 記事要約
	summaryStart := time.Now()
	summary, err := p.geminiRepo.SummarizeURL(ctx, article.Link)
	if err != nil {
		logger.Printf("Error summarizing article %s: %v", article.Title, err)
		return fmt.Errorf("summarizing article: %w", err)
	}
	summaryDuration := time.Since(summaryStart)

	// 3. Hatenaコメント取得と要約
	// 注意: Redditとは異なり、HatenaとLobstersはコメント機能が有効
	// 記事通知とコメント通知を別々に送信する
	var commentSummary *string
	var commentDuration time.Duration
	if err := p.fetchAndProcessHatenaComments(ctx, article, &commentSummary, &commentDuration); err != nil {
		logger.Printf("Warning: Failed to fetch Hatena comments for %s: %v", article.Title, err)
		// Hatenaコメント取得失敗でも処理は続行
	}

	// 5. 通知送信
	slackStart := time.Now()
	// 記事通知
	if err := p.slackRepo.Send(ctx, repository.Notification{
		Title:   article.Title,
		Source:  article.Source,
		URL:     article.Link,
		Summary: summary.Summary,
	}); err != nil {
		logger.Printf("Error sending article notification for %s: %v", article.Title, err)
		return fmt.Errorf("sending article notification: %w", err)
	}

	// コメント通知（コメントがある場合）
	if commentSummary != nil {
		if err := p.slackRepo.Send(ctx, repository.Notification{
			Title:   article.Title + " - コメント",
			Source:  article.Source,
			URL:     article.Link,
			Summary: *commentSummary,
		}); err != nil {
			logger.Printf("Error sending comment notification for %s: %v", article.Title, err)
			return fmt.Errorf("sending comment notification: %w", err)
		}
	}
	slackDuration := time.Since(slackStart)

	// 6. インデックス更新
	processStart := time.Now()
	if err := p.processedRepo.MarkAsProcessed(ctx, article); err != nil {
		logger.Printf("Error marking article as processed %s: %v\nStack:\n%s", article.Title, err, debug.Stack())
		return fmt.Errorf("marking as processed: %w", err)
	}
	processDuration := time.Since(processStart)

	totalDuration := time.Since(start)
	logger.Printf("Article processing completed title=%s total_duration_ms=%d summary_duration_ms=%d slack_duration_ms=%d process_duration_ms=%d",
		article.Title, totalDuration.Milliseconds(), summaryDuration.Milliseconds(), slackDuration.Milliseconds(), processDuration.Milliseconds())

	return nil
}

// fetchAndProcessHatenaComments handles Hatena bookmark comment retrieval and summarization
func (p *HatenaProcessor) fetchAndProcessHatenaComments(ctx context.Context, article repository.Item, commentSummary **string, commentDuration *time.Duration) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)

	logger.Printf("Comments fetching started url=%s", article.Link)
	commentStart := time.Now()

	// Fetch Hatena bookmark comments
	comments, err := p.hatenaRSSRepo.FetchComments(ctx, article.Link)
	if err != nil {
		return fmt.Errorf("failed to fetch Hatena comments: %w", err)
	}

	if comments.Text == "" {
		logger.Printf("No Hatena comments found for %s", article.Title)
		*commentDuration = time.Since(commentStart)
		return nil
	}

	logger.Printf("Comments summarization started text_length=%d", len(comments.Text))

	// Summarize comments
	summaryResponse, err := p.geminiRepo.SummarizeComments(ctx, comments.Text)
	if err != nil {
		return fmt.Errorf("failed to summarize Hatena comments: %w", err)
	}

	*commentSummary = &summaryResponse.Summary
	*commentDuration = time.Since(commentStart)

	logger.Printf("Comments summarization completed summary_length=%d duration_ms=%d",
		len(summaryResponse.Summary), commentDuration.Milliseconds())

	return nil
}
