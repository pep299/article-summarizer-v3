package service

import (
	"context"
	"log"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

type URL struct {
	gemini repository.GeminiRepository
	slack  repository.SlackRepository
}

func NewURL(
	gemini repository.GeminiRepository,
	slack repository.SlackRepository,
) *URL {
	return &URL{
		gemini: gemini,
		slack:  slack,
	}
}

func (u *URL) Process(ctx context.Context, url string) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	startTime := time.Now()

	logger.Printf("On-demand URL processing started url=%s", url)

	// Summarization phase
	summaryStart := time.Now()
	summary, err := u.gemini.SummarizeURLForOnDemand(ctx, url)
	if err != nil {
		logger.Printf("Error summarizing URL %s: %v", url, err)
		return err
	}
	summaryDuration := time.Since(summaryStart)

	article := repository.Item{
		Title:  summary.Title,
		Link:   url,
		Source: "on-demand",
	}

	// Slack notification phase
	slackStart := time.Now()
	// Use the on-demand specific method for Slack notification
	// Note: The targetChannel should be passed from the application layer
	// For now, using the default channel of the slack repository
	if err := u.slack.SendOnDemandSummary(ctx, article, *summary, ""); err != nil {
		logger.Printf("Error sending on-demand Slack summary for URL %s: %v", url, err)
		return err
	}
	slackDuration := time.Since(slackStart)

	totalDuration := time.Since(startTime)
	logger.Printf("On-demand URL processing completed url=%s total_duration_ms=%d summary_duration_ms=%d slack_duration_ms=%d",
		url, totalDuration.Milliseconds(), summaryDuration.Milliseconds(), slackDuration.Milliseconds())
	return nil
}
