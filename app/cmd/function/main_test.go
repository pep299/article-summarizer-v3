package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSummarizeArticles_ProcessEndpoint(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("RSS_FEEDS", `[{"name":"test","url":"http://example.com/feed","channel":"test-channel"}]`)
	os.Setenv("WEBHOOK_AUTH_TOKEN", "test-webhook-token")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("RSS_FEEDS")
		os.Unsetenv("WEBHOOK_AUTH_TOKEN")
	}()

	tests := []struct {
		name           string
		method         string
		path           string
		authHeader     string
		payload        interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid process request",
			method:         "POST",
			path:           "/process",
			authHeader:     "Bearer test-webhook-token",
			payload:        map[string]string{"feedName": "test"},
			expectedStatus: http.StatusInternalServerError, // Will fail due to mock RSS feed
		},
		{
			name:           "Missing auth header",
			method:         "POST",
			path:           "/process",
			authHeader:     "",
			payload:        map[string]string{"feedName": "test"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Missing Authorization header",
		},
		{
			name:           "Invalid auth header format",
			method:         "POST",
			path:           "/process",
			authHeader:     "InvalidToken",
			payload:        map[string]string{"feedName": "test"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Invalid Authorization header format",
		},
		{
			name:           "Invalid token",
			method:         "POST",
			path:           "/process",
			authHeader:     "Bearer wrong-token",
			payload:        map[string]string{"feedName": "test"},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Invalid token",
		},
		{
			name:           "GET method not allowed",
			method:         "GET",
			path:           "/process",
			authHeader:     "Bearer test-webhook-token",
			payload:        nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Missing feedName",
			method:         "POST",
			path:           "/process",
			authHeader:     "Bearer test-webhook-token",
			payload:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'feedName' in payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if tt.payload != nil {
				json.NewEncoder(&body).Encode(tt.payload)
			}

			req := httptest.NewRequest(tt.method, tt.path, &body)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			SummarizeArticles(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestSummarizeArticles_WebhookEndpoint(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("RSS_FEEDS", `[{"name":"test","url":"http://example.com/feed","channel":"test-channel"}]`)
	os.Setenv("WEBHOOK_AUTH_TOKEN", "test-webhook-token")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("RSS_FEEDS")
		os.Unsetenv("WEBHOOK_AUTH_TOKEN")
	}()

	tests := []struct {
		name           string
		method         string
		path           string
		authHeader     string
		payload        interface{}
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Valid webhook request",
			method:         "POST",
			path:           "/webhook",
			authHeader:     "Bearer test-webhook-token",
			payload:        map[string]string{"url": "http://example.com/article"},
			expectedStatus: http.StatusInternalServerError, // Will fail due to mock URL
		},
		{
			name:           "Missing auth header",
			method:         "POST",
			path:           "/webhook",
			authHeader:     "",
			payload:        map[string]string{"url": "http://example.com/article"},
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Missing Authorization header",
		},
		{
			name:           "Invalid token",
			method:         "POST",
			path:           "/webhook",
			authHeader:     "Bearer wrong-token",
			payload:        map[string]string{"url": "http://example.com/article"},
			expectedStatus: http.StatusForbidden,
			expectedBody:   "Invalid token",
		},
		{
			name:           "GET method not allowed",
			method:         "GET",
			path:           "/webhook",
			authHeader:     "Bearer test-webhook-token",
			payload:        nil,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
		{
			name:           "Missing URL",
			method:         "POST",
			path:           "/webhook",
			authHeader:     "Bearer test-webhook-token",
			payload:        map[string]string{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Missing 'url' in payload",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body bytes.Buffer
			if tt.payload != nil {
				json.NewEncoder(&body).Encode(tt.payload)
			}

			req := httptest.NewRequest(tt.method, tt.path, &body)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			SummarizeArticles(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedBody != "" && !strings.Contains(w.Body.String(), tt.expectedBody) {
				t.Errorf("Expected body to contain %q, got %q", tt.expectedBody, w.Body.String())
			}
		})
	}
}

func TestSummarizeArticles_NotFound(t *testing.T) {
	// Set required environment variables for testing
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("RSS_FEEDS", `[{"name":"test","url":"http://example.com/feed","channel":"test-channel"}]`)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("RSS_FEEDS")
	}()

	req := httptest.NewRequest("GET", "/unknown", nil)
	w := httptest.NewRecorder()
	SummarizeArticles(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestMain(t *testing.T) {
	// Test that main function doesn't panic
	// This is a simple test since main() is just a placeholder for Cloud Functions
	main()
}
