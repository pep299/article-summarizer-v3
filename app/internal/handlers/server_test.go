package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

func TestNewServer(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		GeminiModel:         "test-model",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
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
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
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

// Mock implementations for testing
type mockRSSClient struct {
	items []rss.Item
	err   error
}

func (m *mockRSSClient) FetchFeed(ctx context.Context, feedName, url string) ([]rss.Item, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.items, nil
}

type mockGeminiClient struct {
	response *gemini.SummarizeResponse
	err      error
}

func (m *mockGeminiClient) SummarizeURL(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

type mockSlackClient struct {
	sentSummaries []slack.ArticleSummary
	err           error
}

func (m *mockSlackClient) SendArticleSummary(ctx context.Context, summary slack.ArticleSummary) error {
	if m.err != nil {
		return m.err
	}
	m.sentSummaries = append(m.sentSummaries, summary)
	return nil
}

func TestProcessSingleFeed_Success(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
		HatenaRSSURL:        "http://hatena.example.com/rss",
		LobstersRSSURL:      "http://lobsters.example.com/rss",
	}

	mockRSS := &mockRSSClient{
		items: []rss.Item{
			{
				Title:       "Test Article 1",
				Link:        "http://example.com/1",
				Description: "Test description 1",
				Source:      "Test Feed",
			},
		},
	}

	mockGemini := &mockGeminiClient{
		response: &gemini.SummarizeResponse{
			Summary: "Test summary",
		},
	}

	mockSlack := &mockSlackClient{}

	server := NewServerWithDeps(cfg, mockRSS, mockGemini, mockSlack, mockSlack, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that Slack messages were sent
	if len(mockSlack.sentSummaries) == 0 {
		t.Error("Expected Slack summaries to be sent")
	}
}

func TestProcessSingleFeed_RSSError(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:   "test-key",
		SlackBotToken:  "xoxb-test-token",
		SlackChannel:   "#test",
		HatenaRSSURL:   "http://hatena.example.com/rss",
		LobstersRSSURL: "http://lobsters.example.com/rss",
	}

	mockRSS := &mockRSSClient{
		err: errors.New("RSS fetch failed"),
	}

	server := NewServerWithDeps(cfg, mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err == nil {
		t.Error("Expected error from RSS fetch failure")
	}

	if !strings.Contains(err.Error(), "RSS fetch failed") {
		t.Errorf("Expected error to contain 'RSS fetch failed', got: %v", err)
	}
}

func TestProcessSingleFeed_EmptyFeed(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:   "test-key",
		SlackBotToken:  "xoxb-test-token",
		SlackChannel:   "#test",
		HatenaRSSURL:   "http://hatena.example.com/rss",
		LobstersRSSURL: "http://lobsters.example.com/rss",
	}

	mockRSS := &mockRSSClient{
		items: []rss.Item{}, // Empty feed
	}

	server := NewServerWithDeps(cfg, mockRSS, nil, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleFeed(ctx, "hatena")
	if err != nil {
		t.Errorf("Expected no error for empty feed, got: %v", err)
	}
}

func TestProcessSingleURL_WithMocks(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}

	mockGemini := &mockGeminiClient{
		response: &gemini.SummarizeResponse{
			Summary: "Test URL summary",
		},
	}

	mockSlack := &mockSlackClient{}

	server := NewServerWithDeps(cfg, nil, mockGemini, mockSlack, mockSlack, nil)
	ctx := context.Background()

	err := server.ProcessSingleURL(ctx, "http://example.com/article")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Check that webhook Slack message was sent
	if len(mockSlack.sentSummaries) == 0 {
		t.Error("Expected webhook Slack summary to be sent")
	}
}

func TestProcessSingleURL_GeminiError(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}

	mockGemini := &mockGeminiClient{
		err: errors.New("Gemini summarization failed"),
	}

	server := NewServerWithDeps(cfg, nil, mockGemini, nil, nil, nil)
	ctx := context.Background()

	err := server.ProcessSingleURL(ctx, "http://example.com/article")
	if err == nil {
		t.Error("Expected error from Gemini failure")
	}

	if !strings.Contains(err.Error(), "summarizing URL") {
		t.Errorf("Expected error to mention URL summarization, got: %v", err)
	}
}

func TestProcessSingleURL_SlackError(t *testing.T) {
	cfg := &config.Config{
		GeminiAPIKey:        "test-key",
		SlackBotToken:       "xoxb-test-token",
		SlackChannel:        "#test",
		WebhookSlackChannel: "#webhook-test",
	}

	mockGemini := &mockGeminiClient{
		response: &gemini.SummarizeResponse{
			Summary: "Test URL summary",
		},
	}

	mockSlack := &mockSlackClient{
		err: errors.New("Slack webhook failed"),
	}

	server := NewServerWithDeps(cfg, nil, mockGemini, mockSlack, mockSlack, nil)
	ctx := context.Background()

	err := server.ProcessSingleURL(ctx, "http://example.com/article")
	if err == nil {
		t.Error("Expected error from Slack failure")
	}

	if !strings.Contains(err.Error(), "Slack") {
		t.Errorf("Expected error to mention Slack, got: %v", err)
	}
}
