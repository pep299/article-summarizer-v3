package service

import (
	"context"
	"log"
	"time"

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
	startTime := time.Now()
	log.Printf("🔍 オンデマンド記事処理開始: %s", url)

	summary, err := u.gemini.SummarizeURLForOnDemand(ctx, url)
	if err != nil {
		return err
	}

	article := repository.Item{
		Title:  url,
		Link:   url,
		Source: "オンデマンドリクエスト",
	}

	// Use the on-demand specific method for Slack notification
	// Note: The targetChannel should be passed from the application layer
	// For now, using the default channel of the slack repository
	if err := u.slack.SendOnDemandSummary(ctx, article, *summary, ""); err != nil {
		return err
	}

	duration := time.Since(startTime)
	log.Printf("✅ オンデマンド記事処理完了: %s (所要時間: %v)", url, duration)
	return nil
}
