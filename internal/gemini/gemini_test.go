package gemini

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/rss"
)

func TestNewClient(t *testing.T) {
	apiKey := "test-api-key"
	model := "gemini-2.5-flash"
	
	client := NewClient(apiKey, model)
	
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	
	if client.apiKey != apiKey {
		t.Errorf("Expected API key '%s', got '%s'", apiKey, client.apiKey)
	}
	
	if client.model != model {
		t.Errorf("Expected model '%s', got '%s'", model, client.model)
	}
	
	if client.httpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
	
	if client.baseURL == "" {
		t.Error("Expected non-empty base URL")
	}
	
	if !strings.Contains(client.baseURL, "generativelanguage.googleapis.com") {
		t.Errorf("Expected base URL to contain Google API domain, got '%s'", client.baseURL)
	}
}

func TestExtractTextFromHTML(t *testing.T) {
	client := NewClient("test-key", "test-model")
	
	tests := []struct {
		name     string
		html     string
		expected []string
		notExpected []string
	}{
		{
			name: "basic HTML extraction",
			html: `<html><body><h1>Title</h1><p>Content</p></body></html>`,
			expected: []string{"Title", "Content"},
			notExpected: []string{"<h1>", "</p>"},
		},
		{
			name: "script and style removal",
			html: `<html><head><script>alert('test');</script><style>body{color:red;}</style></head><body>Main content</body></html>`,
			expected: []string{"Main content"},
			notExpected: []string{"alert", "color:red"},
		},
		{
			name: "whitespace normalization",
			html: `<html><body><p>Multiple     spaces   and
			
			newlines</p></body></html>`,
			expected: []string{"Multiple spaces and newlines"},
			notExpected: []string{"     ", "\n\n"},
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := client.extractTextFromHTML(test.html)
			
			for _, expected := range test.expected {
				if !strings.Contains(result, expected) {
					t.Errorf("Expected result to contain '%s', got '%s'", expected, result)
				}
			}
			
			for _, notExpected := range test.notExpected {
				if strings.Contains(result, notExpected) {
					t.Errorf("Expected result to not contain '%s', got '%s'", notExpected, result)
				}
			}
		})
	}
}

func TestBuildRSSPrompt(t *testing.T) {
	client := NewClient("test-key", "test-model")
	
	content := "This is test content for RSS prompt generation."
	prompt := client.buildRSSPrompt(content)
	
	if !strings.Contains(prompt, content) {
		t.Error("Expected prompt to contain the input content")
	}
	
	if !strings.Contains(prompt, "1000文字以内") {
		t.Error("Expected prompt to contain RSS-specific length constraint")
	}
	
	if !strings.Contains(prompt, "チーム共有") {
		t.Error("Expected prompt to contain team sharing context")
	}
	
	// Test content truncation
	longContent := strings.Repeat("a", 15000)
	truncatedPrompt := client.buildRSSPrompt(longContent)
	if len(truncatedPrompt) > 11000 { // 10KB content + prompt overhead
		t.Error("Expected long content to be truncated")
	}
}

func TestBuildOnDemandPrompt(t *testing.T) {
	client := NewClient("test-key", "test-model")
	
	content := "This is test content for on-demand prompt generation."
	prompt := client.buildOnDemandPrompt(content)
	
	if !strings.Contains(prompt, content) {
		t.Error("Expected prompt to contain the input content")
	}
	
	if !strings.Contains(prompt, "800-1200文字程度") {
		t.Error("Expected prompt to contain on-demand specific length constraint")
	}
	
	if !strings.Contains(prompt, "チーム共有") {
		t.Error("Expected prompt to contain team sharing context")
	}
	
	// Test content truncation
	longContent := strings.Repeat("b", 15000)
	truncatedPrompt := client.buildOnDemandPrompt(longContent)
	if len(truncatedPrompt) > 11000 { // 10KB content + prompt overhead
		t.Error("Expected long content to be truncated")
	}
}

func TestFetchHTMLErrorHandling(t *testing.T) {
	client := NewClient("test-key", "test-model")
	ctx := context.Background()
	
	// Test invalid URL
	_, err := client.fetchHTML(ctx, "invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}
	
	// Test non-existent URL
	_, err = client.fetchHTML(ctx, "http://nonexistent.example.com")
	if err == nil {
		t.Error("Expected error for non-existent URL")
	}
}

func TestFetchHTMLSuccess(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check User-Agent
		userAgent := r.Header.Get("User-Agent")
		if !strings.Contains(userAgent, "Article Summarizer Bot") {
			t.Errorf("Expected User-Agent to contain 'Article Summarizer Bot', got '%s'", userAgent)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html><body><h1>Test Content</h1></body></html>"))
	}))
	defer server.Close()
	
	client := NewClient("test-key", "test-model")
	ctx := context.Background()
	
	html, err := client.fetchHTML(ctx, server.URL)
	if err != nil {
		t.Fatalf("Failed to fetch HTML: %v", err)
	}
	
	if !strings.Contains(html, "Test Content") {
		t.Error("Expected HTML to contain test content")
	}
}

func TestFetchHTMLStatusError(t *testing.T) {
	// Create test server that returns error status
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()
	
	client := NewClient("test-key", "test-model")
	ctx := context.Background()
	
	_, err := client.fetchHTML(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for 404 status")
	}
	
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to mention status code, got: %v", err)
	}
}

func TestConvertRSSItemToRequest(t *testing.T) {
	item := rss.Item{
		Title:       "Test Article Title",
		Link:        "http://example.com/test",
		Description: "Test article description",
		GUID:        "test-guid-123",
	}
	
	request := ConvertRSSItemToRequest(item)
	
	if request.Title != item.Title {
		t.Errorf("Expected title '%s', got '%s'", item.Title, request.Title)
	}
	
	if request.Link != item.Link {
		t.Errorf("Expected link '%s', got '%s'", item.Link, request.Link)
	}
	
	if request.Description != item.Description {
		t.Errorf("Expected description '%s', got '%s'", item.Description, request.Description)
	}
	
	if request.Content != "" {
		t.Errorf("Expected empty content, got '%s'", request.Content)
	}
}

func TestSummarizeResponseStructure(t *testing.T) {
	// Test that the SummarizeResponse struct is correctly defined
	response := SummarizeResponse{
		Summary:     "Test summary",
		ProcessedAt: time.Now(),
	}
	
	if response.Summary != "Test summary" {
		t.Error("Failed to set Summary field")
	}
	
	if response.ProcessedAt.IsZero() {
		t.Error("Failed to set ProcessedAt field")
	}
}

func TestGeminiRequestStructure(t *testing.T) {
	// Test internal request structures (indirectly)
	client := NewClient("test-key", "test-model")
	
	// Test that the client can build prompts correctly
	prompt := client.buildRSSPrompt("test content")
	if prompt == "" {
		t.Error("Expected non-empty prompt")
	}
	
	onDemandPrompt := client.buildOnDemandPrompt("test content")
	if onDemandPrompt == "" {
		t.Error("Expected non-empty on-demand prompt")
	}
	
	// Verify prompts are different
	if prompt == onDemandPrompt {
		t.Error("Expected RSS and on-demand prompts to be different")
	}
}
