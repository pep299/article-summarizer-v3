package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

// Mock interfaces for testing
type mockRSSClient struct {
	fetchFeedFunc func(ctx context.Context, feedName, url string) ([]rss.Item, error)
}

func (m *mockRSSClient) FetchFeed(ctx context.Context, feedName, url string) ([]rss.Item, error) {
	if m.fetchFeedFunc != nil {
		return m.fetchFeedFunc(ctx, feedName, url)
	}
	return []rss.Item{}, nil
}

type mockGeminiClient struct {
	summarizeURLFunc func(ctx context.Context, url string) (*gemini.SummarizeResponse, error)
}

func (m *mockGeminiClient) SummarizeURL(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
	if m.summarizeURLFunc != nil {
		return m.summarizeURLFunc(ctx, url)
	}
	return &gemini.SummarizeResponse{Summary: "Mock summary"}, nil
}

type mockSlackClient struct {
	sendArticleSummaryFunc func(ctx context.Context, summary slack.ArticleSummary) error
}

func (m *mockSlackClient) SendArticleSummary(ctx context.Context, summary slack.ArticleSummary) error {
	if m.sendArticleSummaryFunc != nil {
		return m.sendArticleSummaryFunc(ctx, summary)
	}
	return nil
}

type mockCacheManager struct {
	isCachedFunc       func(ctx context.Context, item rss.Item) (bool, error)
	markAsProcessedFunc func(ctx context.Context, item rss.Item) error
}

func (m *mockCacheManager) IsCached(ctx context.Context, item rss.Item) (bool, error) {
	if m.isCachedFunc != nil {
		return m.isCachedFunc(ctx, item)
	}
	return false, nil
}

func (m *mockCacheManager) MarkAsProcessed(ctx context.Context, item rss.Item) error {
	if m.markAsProcessedFunc != nil {
		return m.markAsProcessedFunc(ctx, item)
	}
	return nil
}

func (m *mockCacheManager) GetStats(ctx context.Context) (*cache.Stats, error) {
	return nil, nil
}

func (m *mockCacheManager) Clear(ctx context.Context) error {
	return nil
}

func (m *mockCacheManager) Close() error {
	return nil
}

// Helper function to create a test server with mocks
func createTestServer() *Server {
	cfg := &config.Config{
		RSSFeeds: map[string]config.RSSFeedConfig{
			"test-feed": {
				Name:    "Test Feed",
				URL:     "http://example.com/rss",
				Enabled: true,
			},
			"disabled-feed": {
				Name:    "Disabled Feed",
				URL:     "http://example.com/disabled",
				Enabled: false,
			},
		},
	}

	return NewServerWithDeps(cfg, nil, nil, nil, nil)
}

// Helper function to create a test server with specific mocks
func createTestServerWithMocks(rssClient RSSClient, geminiClient GeminiClient, slackClient SlackClient, cacheManager CacheManager) *Server {
	cfg := &config.Config{
		RSSFeeds: map[string]config.RSSFeedConfig{
			"test-feed": {
				Name:    "Test Feed",
				URL:     "http://example.com/rss",
				Enabled: true,
			},
			"disabled-feed": {
				Name:    "Disabled Feed",
				URL:     "http://example.com/disabled",
				Enabled: false,
			},
		},
	}

	return NewServerWithDeps(cfg, rssClient, geminiClient, slackClient, cacheManager)
}

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		CacheType:     "memory",
		CacheDuration: 24,
		GeminiAPIKey:  "test-key",
		GeminiModel:   "test-model",
		SlackBotToken: "xoxb-test-token",
		SlackChannel:  "#test",
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	if server.config != cfg {
		t.Error("Expected config to be set")
	}

	if server.rssClient == nil {
		t.Error("Expected RSS client to be initialized")
	}

	if server.geminiClient == nil {
		t.Error("Expected Gemini client to be initialized")
	}

	if server.slackClient == nil {
		t.Error("Expected Slack client to be initialized")
	}

	if server.cacheManager == nil {
		t.Error("Expected cache manager to be initialized")
	}
}

func TestProcessSingleFeed_NonExistentFeed(t *testing.T) {
	server := createTestServer()
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error for non-existent feed")
	}

	expectedMsg := "feed non-existent not found"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestProcessSingleFeed_DisabledFeed(t *testing.T) {
	server := createTestServer()
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "disabled-feed")
	if err != nil {
		t.Errorf("Expected no error for disabled feed, got: %v", err)
	}
}

func TestProcessSingleFeed_RSSFetchError(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return nil, errors.New("RSS fetch failed")
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "test-feed")
	if err == nil {
		t.Error("Expected error when RSS fetch fails")
	}

	if err.Error() != "fetching RSS feed test-feed: RSS fetch failed" {
		t.Errorf("Expected RSS fetch error to be wrapped, got: %v", err)
	}
}

func TestProcessSingleFeed_EmptyFeed(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{}, nil
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "test-feed")
	if err != nil {
		t.Errorf("Expected no error for empty feed, got: %v", err)
	}
}

func TestProcessSingleFeed_CacheCheckError(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{
				{
					Title: "Test Article",
					Link:  "http://example.com/article1",
					GUID:  "article1",
				},
			}, nil
		},
	}
	mockCache := &mockCacheManager{
		isCachedFunc: func(ctx context.Context, item rss.Item) (bool, error) {
			return false, errors.New("cache check failed")
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, mockCache)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "test-feed")
	if err == nil {
		t.Error("Expected error when cache check fails")
	}

	if err.Error() != "checking cache for article Test Article: cache check failed" {
		t.Errorf("Expected cache check error to be wrapped, got: %v", err)
	}
}

func TestProcessSingleFeed_AllArticlesCached(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{
				{
					Title: "Test Article",
					Link:  "http://example.com/article1",
					GUID:  "article1",
				},
			}, nil
		},
	}
	mockCache := &mockCacheManager{
		isCachedFunc: func(ctx context.Context, item rss.Item) (bool, error) {
			return true, nil // All articles are cached
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, mockCache)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "test-feed")
	if err != nil {
		t.Errorf("Expected no error when all articles are cached, got: %v", err)
	}
}

func TestProcessSingleFeed_SuccessfulProcessing(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{
				{
					Title: "Test Article",
					Link:  "http://example.com/article1",
					GUID:  "article1",
				},
			}, nil
		},
	}
	mockCache := &mockCacheManager{
		isCachedFunc: func(ctx context.Context, item rss.Item) (bool, error) {
			return false, nil // Article not cached
		},
		markAsProcessedFunc: func(ctx context.Context, item rss.Item) error {
			return nil
		},
	}
	mockGemini := &mockGeminiClient{
		summarizeURLFunc: func(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
			return &gemini.SummarizeResponse{Summary: "Test summary"}, nil
		},
	}
	mockSlack := &mockSlackClient{
		sendArticleSummaryFunc: func(ctx context.Context, summary slack.ArticleSummary) error {
			return nil
		},
	}
	server := createTestServerWithMocks(mockRSS, mockGemini, mockSlack, mockCache)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "test-feed")
	if err != nil {
		t.Errorf("Expected successful processing, got: %v", err)
	}
}

func TestProcessArticle_GeminiError(t *testing.T) {
	mockGemini := &mockGeminiClient{
		summarizeURLFunc: func(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
			return nil, errors.New("Gemini API failed")
		},
	}
	server := createTestServerWithMocks(nil, mockGemini, nil, nil)
	ctx := context.Background()

	article := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/article1",
	}

	err := server.processArticle(ctx, article)
	if err == nil {
		t.Error("Expected error when Gemini API fails")
	}

	if err.Error() != "summarizing article: Gemini API failed" {
		t.Errorf("Expected Gemini API error to be wrapped, got: %v", err)
	}
}

func TestProcessArticle_SlackError(t *testing.T) {
	mockGemini := &mockGeminiClient{
		summarizeURLFunc: func(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
			return &gemini.SummarizeResponse{Summary: "Test summary"}, nil
		},
	}
	mockSlack := &mockSlackClient{
		sendArticleSummaryFunc: func(ctx context.Context, summary slack.ArticleSummary) error {
			return errors.New("Slack API failed")
		},
	}
	server := createTestServerWithMocks(nil, mockGemini, mockSlack, nil)
	ctx := context.Background()

	article := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/article1",
	}

	err := server.processArticle(ctx, article)
	if err == nil {
		t.Error("Expected error when Slack API fails")
	}

	if err.Error() != "sending Slack notification: Slack API failed" {
		t.Errorf("Expected Slack API error to be wrapped, got: %v", err)
	}
}

func TestProcessArticle_CacheError(t *testing.T) {
	mockGemini := &mockGeminiClient{
		summarizeURLFunc: func(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
			return &gemini.SummarizeResponse{Summary: "Test summary"}, nil
		},
	}
	mockSlack := &mockSlackClient{
		sendArticleSummaryFunc: func(ctx context.Context, summary slack.ArticleSummary) error {
			return nil
		},
	}
	mockCache := &mockCacheManager{
		markAsProcessedFunc: func(ctx context.Context, item rss.Item) error {
			return errors.New("Cache save failed")
		},
	}
	server := createTestServerWithMocks(nil, mockGemini, mockSlack, mockCache)
	ctx := context.Background()

	article := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/article1",
	}

	err := server.processArticle(ctx, article)
	if err == nil {
		t.Error("Expected error when cache save fails")
	}

	if err.Error() != "caching article: Cache save failed" {
		t.Errorf("Expected cache save error to be wrapped, got: %v", err)
	}
}

func TestProcessArticle_Success(t *testing.T) {
	mockGemini := &mockGeminiClient{
		summarizeURLFunc: func(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
			if url != "http://example.com/article1" {
				t.Errorf("Expected URL 'http://example.com/article1', got '%s'", url)
			}
			return &gemini.SummarizeResponse{Summary: "Test summary"}, nil
		},
	}
	mockSlack := &mockSlackClient{
		sendArticleSummaryFunc: func(ctx context.Context, summary slack.ArticleSummary) error {
			if summary.RSS.Title != "Test Article" {
				t.Errorf("Expected article title 'Test Article', got '%s'", summary.RSS.Title)
			}
			if summary.Summary.Summary != "Test summary" {
				t.Errorf("Expected summary 'Test summary', got '%s'", summary.Summary.Summary)
			}
			return nil
		},
	}
	mockCache := &mockCacheManager{
		markAsProcessedFunc: func(ctx context.Context, item rss.Item) error {
			if item.Title != "Test Article" {
				t.Errorf("Expected cached article title 'Test Article', got '%s'", item.Title)
			}
			return nil
		},
	}
	server := createTestServerWithMocks(nil, mockGemini, mockSlack, mockCache)
	ctx := context.Background()

	article := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/article1",
	}

	err := server.processArticle(ctx, article)
	if err != nil {
		t.Errorf("Expected successful processing, got: %v", err)
	}
}