package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

func TestNewClient(t *testing.T) {
	webhookURL := "https://hooks.slack.com/test"
	channel := "#test-channel"
	
	client := NewClient(webhookURL, channel)
	
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	
	if client.webhookURL != webhookURL {
		t.Errorf("Expected webhook URL '%s', got '%s'", webhookURL, client.webhookURL)
	}
	
	if client.channel != channel {
		t.Errorf("Expected channel '%s', got '%s'", channel, client.channel)
	}
	
	if client.httpClient == nil {
		t.Error("Expected non-nil http client")
	}
}

func TestSendSimpleMessage(t *testing.T) {
	// Create a test server to mock Slack webhook
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		
		contentType := r.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "#test")
	ctx := context.Background()
	
	err := client.SendSimpleMessage(ctx, "Test message")
	if err != nil {
		t.Fatalf("Failed to send simple message: %v", err)
	}
}

func TestSendSimpleMessageError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "#test")
	ctx := context.Background()
	
	err := client.SendSimpleMessage(ctx, "Test message")
	if err == nil {
		t.Error("Expected error for HTTP 500 response")
	}
	
	if !strings.Contains(err.Error(), "unexpected status code: 500") {
		t.Errorf("Expected error message to contain status code, got: %v", err)
	}
}

func TestSendArticleSummary(t *testing.T) {
	// Create a test server to capture the request
	var receivedPayload string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		r.Body.Read(buf)
		receivedPayload = string(buf)
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "#test")
	ctx := context.Background()
	
	articleSummary := ArticleSummary{
		RSS: rss.Item{
			Title:       "Test Article",
			Link:        "http://example.com/test",
			Description: "Test description",
			PubDate:     "Mon, 02 Jan 2006 15:04:05 MST",
			ParsedDate:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		Summary: gemini.SummarizeResponse{
			Summary:   "Test summary",
			KeyPoints: "• Point 1\n• Point 2",
		},
	}
	
	err := client.SendArticleSummary(ctx, articleSummary)
	if err != nil {
		t.Fatalf("Failed to send article summary: %v", err)
	}
	
	// Check that the payload contains expected elements
	if !strings.Contains(receivedPayload, "Test Article") {
		t.Error("Expected payload to contain article title")
	}
	
	if !strings.Contains(receivedPayload, "http://example.com/test") {
		t.Error("Expected payload to contain article link")
	}
	
	if !strings.Contains(receivedPayload, "Test summary") {
		t.Error("Expected payload to contain summary")
	}
}

func TestSendMultipleSummaries(t *testing.T) {
	// Create a test server to count requests
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "#test")
	ctx := context.Background()
	
	summaries := []ArticleSummary{
		{
			RSS: rss.Item{
				Title: "Article 1",
				Link:  "http://example.com/1",
			},
			Summary: gemini.SummarizeResponse{
				Summary: "Summary 1",
			},
		},
		{
			RSS: rss.Item{
				Title: "Article 2",
				Link:  "http://example.com/2",
			},
			Summary: gemini.SummarizeResponse{
				Summary: "Summary 2",
			},
		},
	}
	
	err := client.SendMultipleSummaries(ctx, summaries)
	if err != nil {
		t.Fatalf("Failed to send multiple summaries: %v", err)
	}
	
	// Should send one request per summary
	if requestCount != len(summaries) {
		t.Errorf("Expected %d requests, got %d", len(summaries), requestCount)
	}
}

func TestSendMultipleSummariesWithError(t *testing.T) {
	// Create a test server that fails on the second request
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ok"))
		}
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "#test")
	ctx := context.Background()
	
	summaries := []ArticleSummary{
		{
			RSS: rss.Item{
				Title: "Article 1",
				Link:  "http://example.com/1",
			},
			Summary: gemini.SummarizeResponse{
				Summary: "Summary 1",
			},
		},
		{
			RSS: rss.Item{
				Title: "Article 2",
				Link:  "http://example.com/2",
			},
			Summary: gemini.SummarizeResponse{
				Summary: "Summary 2",
			},
		},
	}
	
	err := client.SendMultipleSummaries(ctx, summaries)
	if err == nil {
		t.Error("Expected error when one of the requests fails")
	}
	
	// Should still have attempted both requests
	if requestCount != 2 {
		t.Errorf("Expected 2 requests attempted, got %d", requestCount)
	}
}

func TestFormatSlackMessage(t *testing.T) {
	client := NewClient("https://test.com", "#test")
	
	articleSummary := ArticleSummary{
		RSS: rss.Item{
			Title:       "Test Article",
			Link:        "http://example.com/test",
			Description: "Test description",
			PubDate:     "Mon, 02 Jan 2006 15:04:05 MST",
			ParsedDate:  time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
		},
		Summary: gemini.SummarizeResponse{
			Summary:   "This is a test summary",
			KeyPoints: "• Point 1\n• Point 2\n• Point 3",
		},
	}
	
	message := client.formatSlackMessage(articleSummary)
	
	// Check that the message contains expected elements
	if !strings.Contains(message.Text, "New Article Summary") {
		t.Error("Expected message text to contain 'New Article Summary'")
	}
	
	if len(message.Attachments) == 0 {
		t.Fatal("Expected at least one attachment")
	}
	
	attachment := message.Attachments[0]
	
	if attachment.Title != "Test Article" {
		t.Errorf("Expected attachment title 'Test Article', got '%s'", attachment.Title)
	}
	
	if attachment.TitleLink != "http://example.com/test" {
		t.Errorf("Expected attachment title link 'http://example.com/test', got '%s'", attachment.TitleLink)
	}
	
	if !strings.Contains(attachment.Text, "This is a test summary") {
		t.Error("Expected attachment text to contain summary")
	}
	
	if !strings.Contains(attachment.Text, "Point 1") {
		t.Error("Expected attachment text to contain key points")
	}
}

func TestInvalidWebhookURL(t *testing.T) {
	client := NewClient("invalid-url", "#test")
	ctx := context.Background()
	
	err := client.SendSimpleMessage(ctx, "Test message")
	if err == nil {
		t.Error("Expected error for invalid webhook URL")
	}
}

func TestTimeoutHandling(t *testing.T) {
	// Create a server that never responds
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Longer than client timeout
	}))
	defer server.Close()
	
	client := NewClient(server.URL, "#test")
	// Set a very short timeout for testing
	client.httpClient.Timeout = 10 * time.Millisecond
	
	ctx := context.Background()
	
	err := client.SendSimpleMessage(ctx, "Test message")
	if err == nil {
		t.Error("Expected timeout error")
	}
}
