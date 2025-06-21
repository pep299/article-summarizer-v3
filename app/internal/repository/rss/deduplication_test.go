package rss

import (
	"context"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

func TestIntraFeedDeduplication(t *testing.T) {
	// RSS repository を作成
	rssRepo := repository.NewRSSRepository()

	// テスト用の重複したURLを含む記事データ
	testArticles := []repository.Item{
		{
			Title:  "記事1",
			Link:   "https://example.com/article1",
			GUID:   "guid1",
			Source: "test",
		},
		{
			Title:  "記事2（重複URL）",
			Link:   "https://example.com/article1", // 同じURL
			GUID:   "guid2",
			Source: "test",
		},
		{
			Title:  "記事3（重複GUID）",
			Link:   "https://example.com/article3",
			GUID:   "guid1", // 同じGUID
			Source: "test",
		},
		{
			Title:  "記事4（ユニーク）",
			Link:   "https://example.com/article4",
			GUID:   "guid4",
			Source: "test",
		},
	}

	// GetUniqueItems でフィルタリング
	uniqueArticles := rssRepo.GetUniqueItems(testArticles)

	// 期待値: 4件→3件（URL重複で1件除外）
	expectedCount := 3
	if len(uniqueArticles) != expectedCount {
		t.Errorf("Expected %d unique articles, got %d", expectedCount, len(uniqueArticles))
	}

	// 重複URLが除外されているかチェック
	urls := make(map[string]int)
	for _, article := range uniqueArticles {
		urls[article.Link]++
	}

	// 同じURLが複数存在しないことを確認
	for url, count := range urls {
		if count > 1 {
			t.Errorf("URL %s appears %d times in unique articles", url, count)
		}
	}
}

func TestRedditFeedDeduplication(t *testing.T) {
	ctx := context.Background()

	// RSS repository を作成
	rssRepo := repository.NewRSSRepository()
	redditRepo := NewRedditRSSRepository(rssRepo)

	// Reddit記事を取得
	articles, err := redditRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Reddit articles: %v", err)
	}

	// 同じ記事を2回取得して重複をチェック
	articles2, err := redditRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Reddit articles second time: %v", err)
	}

	// 両方の結果が同じであることを確認（重複排除されている）
	if len(articles) != len(articles2) {
		t.Errorf("Article count mismatch: first=%d, second=%d", len(articles), len(articles2))
	}

	// 記事内に重複がないことを確認
	urls := make(map[string]bool)
	for _, article := range articles {
		if urls[article.Link] {
			t.Errorf("Duplicate URL found in Reddit feed: %s", article.Link)
		}
		urls[article.Link] = true
	}
}

func TestHatenaFeedDeduplication(t *testing.T) {
	ctx := context.Background()

	// RSS repository を作成
	rssRepo := repository.NewRSSRepository()
	hatenaRepo := NewHatenaRSSRepository(rssRepo)

	// Hatena記事を取得
	articles, err := hatenaRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Hatena articles: %v", err)
	}

	// 記事内に重複がないことを確認
	urls := make(map[string]bool)
	for _, article := range articles {
		if urls[article.Link] {
			t.Errorf("Duplicate URL found in Hatena feed: %s", article.Link)
		}
		urls[article.Link] = true
	}
}

func TestLobstersFeedDeduplication(t *testing.T) {
	ctx := context.Background()

	// RSS repository を作成
	rssRepo := repository.NewRSSRepository()
	lobstersRepo := NewLobstersRSSRepository(rssRepo)

	// Lobsters記事を取得
	articles, err := lobstersRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Lobsters articles: %v", err)
	}

	// 記事内に重複がないことを確認
	urls := make(map[string]bool)
	for _, article := range articles {
		if urls[article.Link] {
			t.Errorf("Duplicate URL found in Lobsters feed: %s", article.Link)
		}
		urls[article.Link] = true
	}
}
