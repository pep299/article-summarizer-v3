package integration

import (
	"context"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/repository/rss"
)

func TestCrossFeedDeduplication(t *testing.T) {
	ctx := context.Background()

	// RSS repository を作成
	rssRepo := repository.NewRSSRepository()

	// 各フィードのリポジトリを作成
	redditRepo := rss.NewRedditRSSRepository(rssRepo)
	hatenaRepo := rss.NewHatenaRSSRepository(rssRepo)
	lobstersRepo := rss.NewLobstersRSSRepository(rssRepo)

	// 各フィードから記事を取得
	redditArticles, err := redditRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Reddit articles: %v", err)
	}

	hatenaArticles, err := hatenaRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Hatena articles: %v", err)
	}

	lobstersArticles, err := lobstersRepo.FetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch Lobsters articles: %v", err)
	}

	// GCS repository を作成してURL正規化をテスト
	processedRepo, err := repository.NewProcessedArticleRepository()
	if err != nil {
		t.Fatalf("Failed to create processed repository: %v", err)
	}
	defer processedRepo.Close()

	// フィード間重複検出テスト
	t.Run("RedditLobstersDuplication", func(t *testing.T) {
		testCrossFeedDuplication(t, processedRepo, redditArticles, lobstersArticles, "Reddit", "Lobsters")
	})

	t.Run("RedditHatenaDuplication", func(t *testing.T) {
		testCrossFeedDuplication(t, processedRepo, redditArticles, hatenaArticles, "Reddit", "Hatena")
	})

	t.Run("LobstersHatenaDuplication", func(t *testing.T) {
		testCrossFeedDuplication(t, processedRepo, lobstersArticles, hatenaArticles, "Lobsters", "Hatena")
	})
}

func testCrossFeedDuplication(t *testing.T, processedRepo repository.ProcessedArticleRepository, articles1, articles2 []repository.Item, feed1, feed2 string) {
	// 1つ目のフィードのURLをマップに格納
	feed1URLs := make(map[string]string)
	for _, article := range articles1 {
		key := processedRepo.GenerateKey(article)
		feed1URLs[key] = article.Title
	}

	// 2つ目のフィードで重複をチェック
	duplicateCount := 0
	var duplicateExamples []string

	for _, article := range articles2 {
		key := processedRepo.GenerateKey(article)
		if title1, exists := feed1URLs[key]; exists {
			duplicateCount++
			if len(duplicateExamples) < 3 {
				duplicateExamples = append(duplicateExamples, article.Title+" ("+feed1+" : "+title1+")")
			}
		}
	}

	t.Logf("%s-%s間の重複: %d件検出", feed1, feed2, duplicateCount)

	if len(duplicateExamples) > 0 {
		t.Logf("重複例:")
		for i, example := range duplicateExamples {
			t.Logf("  %d. %s", i+1, example)
		}
	}

	// 重複が検出されても失敗ではない（情報として記録）
	// 実際のフィード間重複は正常な動作
}

func TestBatchProcessingDeduplication(t *testing.T) {
	ctx := context.Background()

	// GCS repository を作成
	processedRepo, err := repository.NewProcessedArticleRepository()
	if err != nil {
		t.Fatalf("Failed to create processed repository: %v", err)
	}
	defer processedRepo.Close()

	// インデックスを読み込み
	index, err := processedRepo.LoadIndex(ctx)
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	// RSS repository を作成
	rssRepo := repository.NewRSSRepository()

	// 各フィードで複数回処理をテスト
	t.Run("RedditMultipleBatchProcessing", func(t *testing.T) {
		testMultipleBatchProcessing(t, rss.NewRedditRSSRepository(rssRepo), processedRepo, index, "Reddit")
	})

	t.Run("HatenaMultipleBatchProcessing", func(t *testing.T) {
		testMultipleBatchProcessing(t, rss.NewHatenaRSSRepository(rssRepo), processedRepo, index, "Hatena")
	})

	t.Run("LobstersMultipleBatchProcessing", func(t *testing.T) {
		testMultipleBatchProcessing(t, rss.NewLobstersRSSRepository(rssRepo), processedRepo, index, "Lobsters")
	})
}

func testMultipleBatchProcessing(t *testing.T, feedRepo interface{}, processedRepo repository.ProcessedArticleRepository, index map[string]*repository.IndexEntry, feedName string) {
	ctx := context.Background()

	// フィードから記事を取得するための型アサーション
	var fetchArticles func(context.Context) ([]repository.Item, error)

	switch repo := feedRepo.(type) {
	case *rss.RedditRSSRepository:
		fetchArticles = repo.FetchArticles
	case *rss.HatenaRSSRepository:
		fetchArticles = repo.FetchArticles
	case *rss.LobstersRSSRepository:
		fetchArticles = repo.FetchArticles
	default:
		t.Fatalf("Unknown repository type")
	}

	// 1回目の処理
	t.Logf("=== %s 1回目の処理 ===", feedName)
	articles1, err := fetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch %s articles (1st time): %v", feedName, err)
	}

	// 新規記事数をカウント
	newArticles1 := filterNewArticles(articles1, processedRepo, index)
	t.Logf("%s 1回目: 取得 %d件, 新規 %d件", feedName, len(articles1), len(newArticles1))

	// 少し待つ（実際のバッチ処理間隔をシミュレート）
	time.Sleep(100 * time.Millisecond)

	// 2回目の処理
	t.Logf("=== %s 2回目の処理 ===", feedName)
	articles2, err := fetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch %s articles (2nd time): %v", feedName, err)
	}

	// 新規記事数をカウント（同じindexを使用して既存記事をフィルタ）
	newArticles2 := filterNewArticles(articles2, processedRepo, index)
	t.Logf("%s 2回目: 取得 %d件, 新規 %d件", feedName, len(articles2), len(newArticles2))

	// 3回目の処理
	t.Logf("=== %s 3回目の処理 ===", feedName)
	articles3, err := fetchArticles(ctx)
	if err != nil {
		t.Fatalf("Failed to fetch %s articles (3rd time): %v", feedName, err)
	}

	// 新規記事数をカウント
	newArticles3 := filterNewArticles(articles3, processedRepo, index)
	t.Logf("%s 3回目: 取得 %d件, 新規 %d件", feedName, len(articles3), len(newArticles3))

	// 複数回処理での一貫性を検証
	t.Run(feedName+"ConsistencyCheck", func(t *testing.T) {
		// 取得記事数がある程度一貫していることを確認
		// RSS feedの性質上、多少の変動は許容
		if len(articles1) == 0 || len(articles2) == 0 || len(articles3) == 0 {
			t.Errorf("One or more batches returned no articles")
		}

		// 各バッチで重複がないことを確認
		checkNoDuplicatesInBatch(t, articles1, feedName+" 1st batch")
		checkNoDuplicatesInBatch(t, articles2, feedName+" 2nd batch")
		checkNoDuplicatesInBatch(t, articles3, feedName+" 3rd batch")

		// 新規記事数が時間と共に減少する傾向があることを確認
		// （既存のindex内の記事は除外されるため）
		t.Logf("%s 新規記事数の推移: %d → %d → %d", feedName, len(newArticles1), len(newArticles2), len(newArticles3))
	})
}

func filterNewArticles(articles []repository.Item, processedRepo repository.ProcessedArticleRepository, index map[string]*repository.IndexEntry) []repository.Item {
	var newArticles []repository.Item
	for _, article := range articles {
		key := processedRepo.GenerateKey(article)
		if !processedRepo.IsProcessed(key, index) {
			newArticles = append(newArticles, article)
		}
	}
	return newArticles
}

func checkNoDuplicatesInBatch(t *testing.T, articles []repository.Item, batchName string) {
	urls := make(map[string]bool)
	for _, article := range articles {
		if urls[article.Link] {
			t.Errorf("Duplicate URL found in %s: %s", batchName, article.Link)
		}
		urls[article.Link] = true
	}
}

func TestURLNormalizationConsistency(t *testing.T) {
	// GCS repository を作成
	processedRepo, err := repository.NewProcessedArticleRepository()
	if err != nil {
		t.Fatalf("Failed to create processed repository: %v", err)
	}
	defer processedRepo.Close()

	// URL正規化の一貫性をテスト
	testCases := []struct {
		name        string
		urls        []string
		expectedKey string
	}{
		{
			name: "HTTP/HTTPS normalization",
			urls: []string{
				"http://example.com/article",
				"https://example.com/article",
			},
			expectedKey: "https://example.com/article",
		},
		{
			name: "WWW normalization",
			urls: []string{
				"https://www.example.com/article",
				"https://example.com/article",
			},
			expectedKey: "https://example.com/article",
		},
		{
			name: "Query parameter removal",
			urls: []string{
				"https://example.com/article?utm_source=reddit",
				"https://example.com/article?ref=twitter",
				"https://example.com/article",
			},
			expectedKey: "https://example.com/article",
		},
		{
			name: "Case normalization",
			urls: []string{
				"https://EXAMPLE.COM/Article",
				"https://example.com/article",
			},
			expectedKey: "https://example.com/article",
		},
		{
			name: "Trailing slash normalization",
			urls: []string{
				"https://example.com/article/",
				"https://example.com/article",
			},
			expectedKey: "https://example.com/article",
		},
		{
			name: "Fragment removal",
			urls: []string{
				"https://example.com/article#section1",
				"https://example.com/article#section2",
				"https://example.com/article",
			},
			expectedKey: "https://example.com/article",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var keys []string
			for _, url := range tc.urls {
				article := repository.Item{Link: url, Source: "test"}
				key := processedRepo.GenerateKey(article)
				keys = append(keys, key)
			}

			// 全てのキーが同じであることを確認
			for i, key := range keys {
				if key != tc.expectedKey {
					t.Errorf("URL %s generated key %s, expected %s", tc.urls[i], key, tc.expectedKey)
				}
			}

			// 全てのキーが一致することを確認
			firstKey := keys[0]
			for i := 1; i < len(keys); i++ {
				if keys[i] != firstKey {
					t.Errorf("Inconsistent key generation: %s vs %s for URLs %s and %s",
						firstKey, keys[i], tc.urls[0], tc.urls[i])
				}
			}
		})
	}
}

func TestProcessedArticleFiltering(t *testing.T) {
	ctx := context.Background()

	// GCS repository を作成
	processedRepo, err := repository.NewProcessedArticleRepository()
	if err != nil {
		t.Fatalf("Failed to create processed repository: %v", err)
	}
	defer processedRepo.Close()

	// インデックスを読み込み
	index, err := processedRepo.LoadIndex(ctx)
	if err != nil {
		t.Fatalf("Failed to load index: %v", err)
	}

	t.Logf("Loaded index with %d entries", len(index))

	// テスト用記事（一部は既存、一部は新規）
	testArticles := []repository.Item{
		{
			Title:  "Test New Article 1",
			Link:   "https://example.com/new-article-1",
			Source: "test",
		},
		{
			Title:  "Test New Article 2",
			Link:   "https://example.com/new-article-2",
			Source: "test",
		},
	}

	// 既存記事を1つ追加（indexから取得）
	if len(index) > 0 {
		for key, entry := range index {
			testArticles = append(testArticles, repository.Item{
				Title:  entry.Title,
				Link:   entry.URL,
				Source: entry.Source,
			})
			t.Logf("Added existing article for test: %s (key: %s)", entry.Title, key)
			break
		}
	}

	// 新規記事のフィルタリングをテスト
	newArticles := filterNewArticles(testArticles, processedRepo, index)

	t.Logf("Original articles: %d", len(testArticles))
	t.Logf("New articles: %d", len(newArticles))

	// 新規記事数が期待値と一致することを確認
	expectedNewCount := 2 // 2つの新規記事
	if len(newArticles) != expectedNewCount {
		t.Errorf("Expected %d new articles, got %d", expectedNewCount, len(newArticles))
	}

	// 新規記事が正しくフィルタリングされていることを確認
	for _, article := range newArticles {
		key := processedRepo.GenerateKey(article)
		if processedRepo.IsProcessed(key, index) {
			t.Errorf("Article %s should be new but is marked as processed", article.Title)
		}
	}
}
