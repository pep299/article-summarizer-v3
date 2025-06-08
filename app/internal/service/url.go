package service

import (
	"context"
	"log"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/model"
	"github.com/pep299/article-summarizer-v3/internal/repository"
)

type URLService struct {
	summaryRepo repository.GeminiRepository
	slackRepo   repository.SlackRepository
}

func NewURLService(
	summaryRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
) *URLService {
	return &URLService{
		summaryRepo: summaryRepo,
		slackRepo:   slackRepo,
	}
}

func (s *URLService) ProcessURL(ctx context.Context, url string) error {
	startTime := time.Now()
	log.Printf("🔍 オンデマンド記事処理開始: %s", url)

	summary, err := s.summaryRepo.SummarizeURL(ctx, url)
	if err != nil {
		return err
	}

	article := model.Article{
		Title:  url,
		Link:   url,
		Source: "オンデマンドリクエスト",
	}

	articleSummary := model.ArticleSummary{
		Article: article,
		Summary: *summary,
	}

	if err := s.slackRepo.SendArticleSummary(ctx, articleSummary); err != nil {
		return err
	}

	duration := time.Since(startTime)
	log.Printf("✅ オンデマンド記事処理完了: %s (所要時間: %v)", url, duration)
	return nil
}