package service

import (
	"context"
	"log"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
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

	summary, err := u.gemini.SummarizeURL(ctx, url)
	if err != nil {
		return err
	}

	article := rss.Item{
		Title:  url,
		Link:   url,
		Source: "オンデマンドリクエスト",
	}

	articleSummary := slack.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	if err := u.slack.SendArticleSummary(ctx, articleSummary); err != nil {
		return err
	}

	duration := time.Since(startTime)
	log.Printf("✅ オンデマンド記事処理完了: %s (所要時間: %v)", url, duration)
	return nil
}