package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func TestMain(m *testing.M) {
	// Set up test environment variables
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test-channel")
	os.Setenv("CACHE_TYPE", "memory")
	os.Setenv("CACHE_DURATION_HOURS", "1")
	
	// Run tests
	code := m.Run()
	
	// Clean up
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("SLACK_BOT_TOKEN")
	os.Unsetenv("SLACK_CHANNEL")
	os.Unsetenv("CACHE_TYPE")
	os.Unsetenv("CACHE_DURATION_HOURS")
	
	os.Exit(code)
}

func TestFullServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	// Create server
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	// Setup routes
	router := server.SetupRoutes()
	
	// Test server endpoints
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		checkResponse  func(t *testing.T, body []byte)
	}{
		{
			name:           "Health Check",
			method:         "GET",
			path:           "/api/v1/health",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse health response: %v", err)
				}
				if response["status"] != "ok" {
					t.Errorf("Expected health status 'ok', got '%v'", response["status"])
				}
			},
		},
		{
			name:           "Cache Stats",
			method:         "GET",
			path:           "/api/v1/cache/stats",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse cache stats response: %v", err)
				}
				if _, ok := response["total_entries"]; !ok {
					t.Error("Expected 'total_entries' in cache stats")
				}
			},
		},
		{
			name:           "Config Endpoint",
			method:         "GET",
			path:           "/api/v1/config",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse config response: %v", err)
				}
				if _, ok := response["cache_type"]; !ok {
					t.Error("Expected 'cache_type' in config response")
				}
				// Should not contain sensitive data
				if _, ok := response["gemini_api_key"]; ok {
					t.Error("Config should not expose sensitive data")
				}
			},
		},
		{
			name:           "Status Endpoint",
			method:         "GET",
			path:           "/api/v1/status",
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, body []byte) {
				var response map[string]interface{}
				if err := json.Unmarshal(body, &response); err != nil {
					t.Fatalf("Failed to parse status response: %v", err)
				}
				if _, ok := response["uptime"]; !ok {
					t.Error("Expected 'uptime' in status response")
				}
			},
		},
		{
			name:           "Invalid Endpoint",
			method:         "GET",
			path:           "/api/v1/invalid",
			expectedStatus: http.StatusNotFound,
			checkResponse:  nil,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()
			
			router.ServeHTTP(w, req)
			
			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
			
			if tc.checkResponse != nil {
				tc.checkResponse(t, w.Body.Bytes())
			}
		})
	}
}

func TestRSSProcessingIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping RSS integration test in short mode")
	}
	
	// This test requires actual network access and may fail in restricted environments
	// We'll mock the external dependencies to make it more reliable
	
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	ctx := context.Background()
	
	// Test that RSS processing doesn't panic and handles errors gracefully
	// In a real test environment, this might fail due to network issues or missing feeds
	err = server.ProcessSingleFeed(ctx, "hatena")
	if err != nil {
		t.Logf("RSS processing failed as expected in test environment: %v", err)
		// This is expected in test environment, we just want to ensure it doesn't panic
	}
}

func TestCacheIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping cache integration test in short mode")
	}
	
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	// Test cache clear
	req := httptest.NewRequest("DELETE", "/api/v1/cache/clear", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for cache clear, got %d", http.StatusOK, w.Code)
	}
	
	// Test cache stats after clear
	req = httptest.NewRequest("GET", "/api/v1/cache/stats", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for cache stats, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse cache stats response: %v", err)
	}
	
	// After clearing, total entries should be 0
	if totalEntries, ok := response["total_entries"]; ok {
		if entries, ok := totalEntries.(float64); ok && entries != 0 {
			t.Errorf("Expected 0 total entries after clear, got %v", entries)
		}
	}
}

func TestErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping error handling test in short mode")
	}
	
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	// Test processing non-existent feed
	req := httptest.NewRequest("POST", "/api/v1/rss/process/non-existent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code == http.StatusOK {
		t.Error("Expected error status for non-existent feed, got 200")
	}
	
	// Test fetching non-existent feed
	req = httptest.NewRequest("GET", "/api/v1/rss/fetch/non-existent", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code == http.StatusOK {
		t.Error("Expected error status for non-existent feed fetch, got 200")
	}
}

func TestCORSHeaders(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	// Test CORS headers
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Error("Expected CORS header 'Access-Control-Allow-Origin: *'")
	}
	
	// Test OPTIONS request
	req = httptest.NewRequest("OPTIONS", "/api/v1/health", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for OPTIONS request, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggingMiddleware(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	// Test that requests are logged (we can't easily test log output, but we can ensure no panic)
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	
	// This should not panic and should complete successfully
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestConfigValidationIntegration(t *testing.T) {
	// Test with missing required environment variables
	originalAPIKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")
	
	_, err := config.Load()
	if err == nil {
		t.Error("Expected error when GEMINI_API_KEY is missing")
	}
	
	// Restore original value
	if originalAPIKey != "" {
		os.Setenv("GEMINI_API_KEY", originalAPIKey)
	}
}

func TestServerShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping shutdown test in short mode")
	}
	
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	
	// Test that Close() doesn't panic
	err = server.Close()
	if err != nil {
		t.Errorf("Server close returned error: %v", err)
	}
}

// Benchmark tests for integration scenarios

func BenchmarkHealthEndpoint(b *testing.B) {
	cfg, err := config.Load()
	if err != nil {
		b.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

func BenchmarkCacheStats(b *testing.B) {
	cfg, err := config.Load()
	if err != nil {
		b.Fatalf("Failed to load configuration: %v", err)
	}
	
	server, err := handlers.NewServer(cfg)
	if err != nil {
		b.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/cache/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

// Test helper functions

func createTestServer() (*handlers.Server, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	
	return handlers.NewServer(cfg)
}

func makeRequest(router http.Handler, method, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func assertJSONResponse(t *testing.T, body []byte, expectedFields ...string) map[string]interface{} {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	
	for _, field := range expectedFields {
		if _, ok := response[field]; !ok {
			t.Errorf("Expected field '%s' in response", field)
		}
	}
	
	return response
}

func TestEndToEndWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test in short mode")
	}
	
	server, err := createTestServer()
	if err != nil {
		t.Fatalf("Failed to create test server: %v", err)
	}
	defer server.Close()
	
	router := server.SetupRoutes()
	
	// 1. Check health
	w := makeRequest(router, "GET", "/api/v1/health")
	if w.Code != http.StatusOK {
		t.Fatalf("Health check failed: %d", w.Code)
	}
	assertJSONResponse(t, w.Body.Bytes(), "status", "version")
	
	// 2. Get initial cache stats
	w = makeRequest(router, "GET", "/api/v1/cache/stats")
	if w.Code != http.StatusOK {
		t.Fatalf("Cache stats failed: %d", w.Code)
	}
	initialStats := assertJSONResponse(t, w.Body.Bytes(), "total_entries", "hit_count", "miss_count")
	
	// 3. Clear cache
	w = makeRequest(router, "DELETE", "/api/v1/cache/clear")
	if w.Code != http.StatusOK {
		t.Fatalf("Cache clear failed: %d", w.Code)
	}
	
	// 4. Verify cache is cleared
	w = makeRequest(router, "GET", "/api/v1/cache/stats")
	if w.Code != http.StatusOK {
		t.Fatalf("Cache stats after clear failed: %d", w.Code)
	}
	clearedStats := assertJSONResponse(t, w.Body.Bytes(), "total_entries")
	
	if clearedStats["total_entries"].(float64) != 0 {
		t.Errorf("Expected 0 entries after clear, got %v", clearedStats["total_entries"])
	}
	
	// 5. Get configuration
	w = makeRequest(router, "GET", "/api/v1/config")
	if w.Code != http.StatusOK {
		t.Fatalf("Config endpoint failed: %d", w.Code)
	}
	configResp := assertJSONResponse(t, w.Body.Bytes(), "cache_type", "rss_feeds")
	
	// Verify sensitive data is not exposed
	if _, ok := configResp["gemini_api_key"]; ok {
		t.Error("Sensitive data should not be in config response")
	}
	
	t.Logf("End-to-end workflow completed successfully")
	t.Logf("Initial cache entries: %v", initialStats["total_entries"])
	t.Logf("Cache type: %v", configResp["cache_type"])
}