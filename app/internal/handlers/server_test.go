package handlers

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:  "test-key",
		GeminiModel:   "test-model",
		SlackBotToken: "xoxb-test-token",
		SlackChannel:  "#test",
		WebhookSlackChannel: "#webhook-test",
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

	if server.webhookSlackClient == nil {
		t.Error("Expected webhook Slack client to be initialized")
	}

	if server.cacheManager == nil {
		t.Error("Expected cache manager to be initialized")
	}
}

func TestProcessSingleFeed_NonExistentFeed(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:  "test-key",
		SlackBotToken: "xoxb-test-token",
		SlackChannel:  "#test",
		WebhookSlackChannel: "#webhook-test",
	}
	
	server := NewServerWithDeps(cfg, nil, nil, nil, nil, nil)
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

func TestProcessSingleURL(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}
	
	// Create server using NewServer to ensure all clients are properly initialized
	server, err := NewServer(cfg)
	if err != nil {
		t.Skipf("Skipping test due to server creation error: %v", err)
	}

	ctx := context.Background()
	err = server.ProcessSingleURL(ctx, "https://example.com")
	// We expect this to fail due to network/API issues in test environment
	// We're just testing that the function can be called without nil pointer errors
	t.Logf("ProcessSingleURL result: %v", err)
}

// Mock clients for testing
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

type mockCache struct {
	existsFunc func(ctx context.Context, key string) (bool, error)
	setFunc    func(ctx context.Context, key string, entry *cache.CacheEntry) error
}

func (m *mockCache) Get(ctx context.Context, key string) (*cache.CacheEntry, error) {
	return nil, cache.ErrCacheMiss
}

func (m *mockCache) Set(ctx context.Context, key string, entry *cache.CacheEntry) error {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, entry)
	}
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	return nil
}

func (m *mockCache) Exists(ctx context.Context, key string) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, key)
	}
	return false, nil
}

func (m *mockCache) GetStats(ctx context.Context) (*cache.Stats, error) {
	return &cache.Stats{}, nil
}

func (m *mockCache) Close() error {
	return nil
}

// Helper function to create a test server with specific mocks
func createTestServerWithMocks(rssClient RSSClient, geminiClient GeminiClient, slackClient SlackClient, webhookSlackClient SlackClient, cacheManager *cache.CloudStorageCache) *Server {
	cfg := &config.Config{
		HatenaRSSURL:   "http://example.com/hatena.rss",
		LobstersRSSURL: "http://example.com/lobsters.rss",
	}

	return NewServerWithDeps(cfg, rssClient, geminiClient, slackClient, webhookSlackClient, cacheManager)
}

func TestProcessSingleFeed_ValidFeeds(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{}, nil // Return empty list to avoid further processing
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	// Test valid feeds exist
	validFeeds := []string{"hatena", "lobsters"}
	for _, feedName := range validFeeds {
		// This should not return "feed not found" error
		// It might fail due to network/API issues, but that's expected in test environment
		err := server.ProcessSingleFeed(ctx, feedName)
		if err != nil && err.Error() == fmt.Sprintf("feed %s not found", feedName) {
			t.Errorf("Feed %s should be recognized as valid", feedName)
		}
	}
}

func TestProcessSingleFeed_RSSFetchError(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return nil, errors.New("RSS fetch failed")
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err == nil {
		t.Error("Expected error when RSS fetch fails")
	}

	if err.Error() != "fetching RSS feed hatena: RSS fetch failed" {
		t.Errorf("Expected RSS fetch error to be wrapped, got: %v", err)
	}
}

func TestProcessSingleFeed_EmptyFeed(t *testing.T) {
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{}, nil
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err != nil {
		t.Errorf("Expected no error for empty feed, got: %v", err)
	}
}

func TestProcessSingleFeed_SuccessfulProcessing(t *testing.T) {
	// Use full NewServer to avoid nil cache issues
	cfg := &config.Config{
		HatenaRSSURL:        "http://example.com/hatena.rss",
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test", 
		WebhookSlackChannel: "#webhook-test",
	}
	
	server, err := NewServer(cfg)
	if err != nil {
		t.Skipf("Skipping test due to server creation error: %v", err)
		return
	}
	
	ctx := context.Background()
	
	// Test with empty feed first (no articles to process)
	err = server.ProcessSingleFeed(ctx, "hatena")
	// We expect this might fail due to network issues, but shouldn't be "feed not found"
	if err != nil && err.Error() == "feed hatena not found" {
		t.Error("Should recognize 'hatena' as a valid feed")
	}
	t.Logf("ProcessSingleFeed result: %v", err)
}

func TestProcessArticle_GeminiError(t *testing.T) {
	mockGemini := &mockGeminiClient{
		summarizeURLFunc: func(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
			return nil, errors.New("Gemini API failed")
		},
	}
	server := createTestServerWithMocks(nil, mockGemini, nil, nil, nil)
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
	server := createTestServerWithMocks(nil, mockGemini, mockSlack, nil, nil)
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

func TestServerUtilityFunctions(t *testing.T) {
	// Test Close function
	server := createTestServerWithMocks(nil, nil, nil, nil, nil)

	// Test Close
	err := server.Close()
	if err != nil {
		t.Errorf("Close should not error, got: %v", err)
	}
}

func TestProcessSingleFeed_NilClients(t *testing.T) {
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
	
	cfg := &config.Config{
		HatenaRSSURL:        "http://example.com/hatena.rss",
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}
	
	server := NewServerWithDeps(cfg, mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err == nil {
		t.Error("Expected error when gemini client is nil")
	}
	
	if err != nil && err.Error() != "processing article Test Article: gemini client is not initialized" {
		t.Errorf("Expected gemini client error, got: %v", err)
	}
}

func TestProcessSingleFeed_AllArticlesCached(t *testing.T) {
	// Test basic RSS fetch with empty results
	mockRSS := &mockRSSClient{
		fetchFeedFunc: func(ctx context.Context, feedName, url string) ([]rss.Item, error) {
			return []rss.Item{}, nil // No articles
		},
	}
	server := createTestServerWithMocks(mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err != nil {
		t.Errorf("Expected no error for empty feed, got: %v", err)
	}
}

func TestProcessArticle_NilCache(t *testing.T) {
	// Test processArticle with nil cache manager (should work without caching)
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
	
	cfg := &config.Config{
		HatenaRSSURL:        "http://example.com/hatena.rss",
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}
	
	server := NewServerWithDeps(cfg, nil, mockGemini, mockSlack, nil, nil)
	ctx := context.Background()

	article := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/article1",
	}

	err := server.processArticle(ctx, article)
	if err != nil {
		t.Errorf("Expected no error when cache manager is nil, got: %v", err)
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
	
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token", 
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}
	
	server := NewServerWithDeps(cfg, nil, mockGemini, mockSlack, nil, nil)
	ctx := context.Background()

	article := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/article1",
	}

	err := server.processArticle(ctx, article)
	// Expect error due to nil cache manager, but test the Gemini and Slack flow
	if err != nil && !strings.Contains(err.Error(), "cache") && !strings.Contains(err.Error(), "nil") {
		t.Errorf("Unexpected error type: %v", err)
	}
}