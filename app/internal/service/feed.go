package service

import (
	"context"
	"fmt"
	"log"

	"github.com/pep299/article-summarizer-v3/internal/model"
	"github.com/pep299/article-summarizer-v3/internal/repository"
)

type FeedService struct {
	rssRepo     repository.RSSRepository
	cacheRepo   repository.CacheRepository
	summaryRepo repository.GeminiRepository
	slackRepo   repository.SlackRepository
}

func NewFeedService(
	rssRepo repository.RSSRepository,
	cacheRepo repository.CacheRepository,
	summaryRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
) *FeedService {
	return &FeedService{
		rssRepo:     rssRepo,
		cacheRepo:   cacheRepo,
		summaryRepo: summaryRepo,
		slackRepo:   slackRepo,
	}
}

func (s *FeedService) ProcessFeed(ctx context.Context, feedName string) error {
	feed, exists := model.Feeds[feedName]
	if !exists {
		return fmt.Errorf("feed %s not found", feedName)
	}

	log.Printf("📡 RSS取得開始: %s", feed.DisplayName)

	articles, err := s.rssRepo.FetchFeed(ctx, feed.DisplayName, feed.URL)
	if err != nil {
		return fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	log.Printf("📄 %d件の記事を取得: %s", len(articles), feed.DisplayName)

	if len(articles) == 0 {
		log.Printf("📋 新着記事はありませんでした: %s", feed.DisplayName)
		return nil
	}

	uncachedArticles := []model.Article{}
	for _, article := range articles {
		if s.cacheRepo != nil {
			cached, err := s.cacheRepo.IsCached(ctx, article)
			if err != nil {
				return fmt.Errorf("checking cache for article %s: %w", article.Title, err)
			}
			if !cached {
				uncachedArticles = append(uncachedArticles, article)
			}
		} else {
			uncachedArticles = append(uncachedArticles, article)
		}
	}

	log.Printf("Processing %d new articles from %s", len(uncachedArticles), feed.DisplayName)

	if len(uncachedArticles) == 0 {
		log.Printf("✅ 新着記事はありません: %s", feed.DisplayName)
		return nil
	}

	for _, article := range uncachedArticles {
		if err := s.processArticle(ctx, article); err != nil {
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
	}

	log.Printf("🎉 %s処理完了: %d件成功", feed.DisplayName, len(uncachedArticles))
	return nil
}

func (s *FeedService) processArticle(ctx context.Context, article model.Article) error {
	log.Printf("🔍 記事処理開始: %s", article.Title)

	summary, err := s.summaryRepo.SummarizeURL(ctx, article.Link)
	if err != nil {
		return fmt.Errorf("summarizing article: %w", err)
	}

	articleSummary := model.ArticleSummary{
		Article: article,
		Summary: *summary,
	}

	if err := s.slackRepo.SendArticleSummary(ctx, articleSummary); err != nil {
		return fmt.Errorf("sending Slack notification: %w", err)
	}

	if s.cacheRepo != nil {
		if err := s.cacheRepo.MarkAsProcessed(ctx, article); err != nil {
			return fmt.Errorf("caching article: %w", err)
		}
	}

	log.Printf("✅ 記事処理完了: %s", article.Title)
	return nil
}