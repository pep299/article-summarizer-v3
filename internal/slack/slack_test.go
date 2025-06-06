package slack

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

func TestNewClient(t *testing.T) {
	botToken := "xoxb-test-token"
	channel := "#test-channel"

	client := NewClient(botToken, channel)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.botToken != botToken {
		t.Errorf("Expected bot token '%s', got '%s'", botToken, client.botToken)
	}

	if client.channel != channel {
		t.Errorf("Expected channel '%s', got '%s'", channel, client.channel)
	}

	if client.httpClient == nil {
		t.Error("Expected non-nil http client")
	}
}

func TestSendArticleSummary(t *testing.T) {
	// Create a test server to mock Slack API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}

		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Authorization header with Bearer token, got '%s'", authHeader)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer server.Close()

	// Note: We can't easily test the actual API call without modifying the implementation
	// to accept a custom base URL. For now, test the basic functionality.
	client := NewClient("xoxb-test-token", "#test")
	_ = context.Background()

	articleSummary := ArticleSummary{
		RSS: rss.Item{
			Title:       "Test Article",
			Link:        "http://example.com/test",
			Description: "Test description",
			Source:      "test-source",
		},
		Summary: gemini.SummarizeResponse{
			Summary:     "Test summary",
			ProcessedAt: time.Now(),
		},
	}

	// Test message formatting
	message := client.formatRSSMessage(articleSummary)
	if !strings.Contains(message, "Test Article") {
		t.Error("Expected message to contain article title")
	}

	if !strings.Contains(message, "http://example.com/test") {
		t.Error("Expected message to contain article link")
	}

	if !strings.Contains(message, "Test summary") {
		t.Error("Expected message to contain summary")
	}
}

func TestSendOnDemandSummary(t *testing.T) {
	client := NewClient("xoxb-test-token", "#test")

	article := rss.Item{
		Title: "On-demand Article",
		Link:  "http://example.com/ondemand",
	}

	summary := gemini.SummarizeResponse{
		Summary:     "On-demand summary",
		ProcessedAt: time.Now(),
	}

	// Test message formatting
	message := client.formatOnDemandMessage(article, summary)
	if !strings.Contains(message, "On-demand Article") {
		t.Error("Expected message to contain article title")
	}

	if !strings.Contains(message, "http://example.com/ondemand") {
		t.Error("Expected message to contain article link")
	}

	if !strings.Contains(message, "On-demand summary") {
		t.Error("Expected message to contain summary")
	}

	if !strings.Contains(message, "オンデマンドAPI") {
		t.Error("Expected message to contain on-demand indicator")
	}
}

func TestFormatRSSMessage(t *testing.T) {
	client := NewClient("xoxb-test-token", "#test")

	articleSummary := ArticleSummary{
		RSS: rss.Item{
			Title:       "Test Article Title",
			Link:        "http://example.com/test",
			Description: "Test description",
			Source:      "test-source",
		},
		Summary: gemini.SummarizeResponse{
			Summary:     "This is a test summary",
			ProcessedAt: time.Now(),
		},
	}

	message := client.formatRSSMessage(articleSummary)

	// Check that the message contains expected elements
	if !strings.Contains(message, "新しい記事を要約しました") {
		t.Error("Expected message to contain RSS summary header")
	}

	if !strings.Contains(message, "Test Article Title") {
		t.Error("Expected message to contain article title")
	}

	if !strings.Contains(message, "http://example.com/test") {
		t.Error("Expected message to contain article link")
	}

	if !strings.Contains(message, "This is a test summary") {
		t.Error("Expected message to contain summary")
	}

	if !strings.Contains(message, "test-source") {
		t.Error("Expected message to contain source")
	}
}

func TestFormatOnDemandMessage(t *testing.T) {
	client := NewClient("xoxb-test-token", "#test")

	article := rss.Item{
		Title: "On-demand Test Article",
		Link:  "http://example.com/ondemand-test",
	}

	summary := gemini.SummarizeResponse{
		Summary:     "This is an on-demand summary",
		ProcessedAt: time.Now(),
	}

	message := client.formatOnDemandMessage(article, summary)

	// Check that the message contains expected elements
	if !strings.Contains(message, "オンデマンド要約リクエスト完了") {
		t.Error("Expected message to contain on-demand header")
	}

	if !strings.Contains(message, "On-demand Test Article") {
		t.Error("Expected message to contain article title")
	}

	if !strings.Contains(message, "http://example.com/ondemand-test") {
		t.Error("Expected message to contain article link")
	}

	if !strings.Contains(message, "This is an on-demand summary") {
		t.Error("Expected message to contain summary")
	}

	if !strings.Contains(message, "オンデマンドAPI") {
		t.Error("Expected message to contain method indicator")
	}
}

func TestFormatOnDemandMessageEmptyTitle(t *testing.T) {
	client := NewClient("xoxb-test-token", "#test")

	article := rss.Item{
		Title: "", // Empty title
		Link:  "http://example.com/no-title",
	}

	summary := gemini.SummarizeResponse{
		Summary:     "Summary for article with no title",
		ProcessedAt: time.Now(),
	}

	message := client.formatOnDemandMessage(article, summary)

	// Should use default title when title is empty
	if !strings.Contains(message, "タイトル取得中...") {
		t.Error("Expected message to contain default title for empty title")
	}
}
