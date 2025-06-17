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
	return &HatenaProcessor{
		hatenaRepo:    rss.NewHatenaRSSRepository(rssRepo),
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
		if err := p.processStandardArticle(ctx, article); err != nil {
			logger.Printf("Error processing article %s: %v", article.Title, err)
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
		logger.Printf("Article processed %d/%d title=%s", i+1, len(limitedArticles), article.Title)
	}

	logger.Printf("Feed processing completed feed=hatena processed_count=%d", len(limitedArticles))
	return nil
}

// processStandardArticle handles articles without comments (Hatena, Lobsters)
func (p *HatenaProcessor) processStandardArticle(ctx context.Context, article repository.Item) error {
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

	// 5. 通知送信
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
