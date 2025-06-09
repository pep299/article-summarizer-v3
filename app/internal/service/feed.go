package service

import (
	"context"
	"fmt"
	"log"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

type Feed struct {
	rss     repository.RSSRepository
	cache   repository.CacheRepository
	gemini  repository.GeminiRepository
	slack   repository.SlackRepository
}

func NewFeed(
	rss repository.RSSRepository,
	cache repository.CacheRepository,
	gemini repository.GeminiRepository,
	slack repository.SlackRepository,
) *Feed {
	return &Feed{
		rss:    rss,
		cache:  cache,
		gemini: gemini,
		slack:  slack,
	}
}

func (f *Feed) Process(ctx context.Context, feedName, feedURL, displayName string) error {
	log.Printf("📡 RSS取得開始: %s", displayName)

	articles, err := f.rss.FetchFeed(ctx, displayName, feedURL)
	if err != nil {
		return fmt.Errorf("fetching RSS feed %s: %w", feedName, err)
	}

	log.Printf("📄 %d件の記事を取得: %s", len(articles), displayName)

	if len(articles) == 0 {
		log.Printf("📋 新着記事はありませんでした: %s", displayName)
		return nil
	}

	// 重複除去とフィルタリング
	uniqueArticles := rss.GetUniqueItems(articles)
	filteredArticles := rss.FilterItems(uniqueArticles)
	
	log.Printf("After filtering: %d articles remain from %s", len(filteredArticles), displayName)

	// キャッシュチェック
	var uncachedArticles []rss.Item
	for _, article := range filteredArticles {
		if f.cache != nil {
			cached, err := f.cache.IsCached(ctx, article)
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

	log.Printf("Processing %d new articles from %s", len(uncachedArticles), displayName)

	if len(uncachedArticles) == 0 {
		log.Printf("✅ 新着記事はありません: %s", displayName)
		return nil
	}

	// 各記事を処理
	for _, article := range uncachedArticles {
		if err := f.processArticle(ctx, article); err != nil {
			return fmt.Errorf("processing article %s: %w", article.Title, err)
		}
	}

	log.Printf("🎉 %s処理完了: %d件成功", displayName, len(uncachedArticles))
	return nil
}

func (f *Feed) processArticle(ctx context.Context, article rss.Item) error {
	log.Printf("🔍 記事処理開始: %s", article.Title)

	summary, err := f.gemini.SummarizeURL(ctx, article.Link)
	if err != nil {
		return fmt.Errorf("summarizing article: %w", err)
	}

	articleSummary := slack.ArticleSummary{
		RSS:     article,
		Summary: *summary,
	}

	if err := f.slack.SendArticleSummary(ctx, articleSummary); err != nil {
		return fmt.Errorf("sending Slack notification: %w", err)
	}

	if f.cache != nil {
		if err := f.cache.MarkAsProcessed(ctx, article); err != nil {
			return fmt.Errorf("caching article: %w", err)
		}
	}

	log.Printf("✅ 記事処理完了: %s", article.Title)
	return nil
}