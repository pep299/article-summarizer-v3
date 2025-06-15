package repository

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestXClient_IsSupported(t *testing.T) {
	client := NewXClient()

	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "valid x.com URL",
			url:      "https://x.com/user/status/123456789",
			expected: true,
		},
		{
			name:     "valid twitter.com URL",
			url:      "https://twitter.com/user/status/123456789",
			expected: true,
		},
		{
			name:     "valid x.com URL with www",
			url:      "https://www.x.com/user/status/123456789",
			expected: true,
		},
		{
			name:     "invalid URL - no status",
			url:      "https://x.com/user",
			expected: false,
		},
		{
			name:     "invalid URL - different domain",
			url:      "https://facebook.com/post/123",
			expected: false,
		},
		{
			name:     "invalid URL - http",
			url:      "http://x.com/user/status/123",
			expected: true, // Should still work with http
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.IsSupported(tt.url)
			if result != tt.expected {
				t.Errorf("IsSupported(%s) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}

func TestXClient_FetchPost_UnsupportedURL(t *testing.T) {
	client := NewXClient()
	ctx := context.Background()

	_, err := client.FetchPost(ctx, "https://facebook.com/post/123")
	if err == nil {
		t.Error("Expected error for unsupported URL, got nil")
	}

	if err.Error() != "unsupported URL format: https://facebook.com/post/123" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestXClient_FetchPost_MockOEmbedAPI(t *testing.T) {
	// Create a mock oEmbed API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Mock oEmbed response
		response := `{
			"url": "https://twitter.com/testuser/status/123456789",
			"author_name": "Test User",
			"author_url": "https://twitter.com/testuser",
			"html": "<blockquote><p>This is a test tweet content</p>&mdash; Test User (@testuser) <a href=\"https://twitter.com/testuser/status/123456789\">December 1, 2023</a></blockquote>",
			"width": 550,
			"height": null,
			"type": "rich",
			"cache_age": "3153600000",
			"provider_name": "Twitter",
			"provider_url": "https://twitter.com",
			"version": "1.0"
		}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Create client with mock URL
	client := &XClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		oembedURL:  mockServer.URL,
	}

	ctx := context.Background()
	postData, err := client.FetchPost(ctx, "https://twitter.com/testuser/status/123456789")
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if postData.AuthorName != "Test User" {
		t.Errorf("Expected AuthorName 'Test User', got '%s'", postData.AuthorName)
	}

	if postData.AuthorURL != "https://twitter.com/testuser" {
		t.Errorf("Expected AuthorURL 'https://twitter.com/testuser', got '%s'", postData.AuthorURL)
	}

	if postData.Text != "This is a test tweet content" {
		t.Errorf("Expected Text 'This is a test tweet content', got '%s'", postData.Text)
	}

	if postData.URL != "https://twitter.com/testuser/status/123456789" {
		t.Errorf("Expected URL 'https://twitter.com/testuser/status/123456789', got '%s'", postData.URL)
	}
}

func TestXClient_FetchPost_OEmbedAPI_Error(t *testing.T) {
	// Create a mock server that returns 404
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer mockServer.Close()

	// Create client with mock URL
	client := &XClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		oembedURL:  mockServer.URL,
	}

	ctx := context.Background()
	_, err := client.FetchPost(ctx, "https://twitter.com/testuser/status/123456789")

	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}

	if err.Error() != "oEmbed API call failed: oEmbed API returned status 404" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestXClient_cleanHTML(t *testing.T) {
	client := NewXClient()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "remove HTML tags",
			input:    "<p>Hello <strong>world</strong></p>",
			expected: "Hello world",
		},
		{
			name:     "decode HTML entities",
			input:    "Hello &amp; goodbye &lt;test&gt;",
			expected: "Hello & goodbye <test>",
		},
		{
			name:     "mixed HTML and entities",
			input:    "<p>Hello &amp; <em>world</em> &quot;test&quot;</p>",
			expected: "Hello & world \"test\"",
		},
		{
			name:     "no HTML or entities",
			input:    "Plain text",
			expected: "Plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := client.cleanHTML(tt.input)
			if result != tt.expected {
				t.Errorf("cleanHTML(%s) = %s, expected %s", tt.input, result, tt.expected)
			}
		})
	}
}
