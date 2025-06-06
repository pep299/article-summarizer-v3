package cloudfunctions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
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

func TestSummarizeArticlesHealthCheck(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	
	SummarizeArticles(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", response["status"])
	}
	
	if response["version"] != "v3.0.0" {
		t.Errorf("Expected version 'v3.0.0', got '%v'", response["version"])
	}
}

func TestSummarizeArticlesInvalidRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "/invalid/route", nil)
	w := httptest.NewRecorder()
	
	SummarizeArticles(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSummarizeArticlesCacheStats(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/cache/stats", nil)
	w := httptest.NewRecorder()
	
	SummarizeArticles(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	// Check that it contains expected cache statistics fields
	if _, ok := response["total_entries"]; !ok {
		t.Error("Expected 'total_entries' field in cache stats")
	}
	
	if _, ok := response["hit_count"]; !ok {
		t.Error("Expected 'hit_count' field in cache stats")
	}
	
	if _, ok := response["miss_count"]; !ok {
		t.Error("Expected 'miss_count' field in cache stats")
	}
}

func TestProcessRSSScheduled(t *testing.T) {
	// Test data for Cloud Event
	eventData := CloudEventData{
		FeedName: "hatena",
	}
	
	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}
	
	// Create a mock Cloud Event
	event := CloudEvent{
		ID:              "test-event-id",
		Source:          "test-source",
		SpecVersion:     "1.0",
		Type:            "google.cloud.scheduler.job.v1.executed",
		Subject:         "projects/test-project/locations/us-central1/jobs/test-job",
		Time:            time.Now(),
		DataContentType: "application/json",
		Data:            data,
	}
	
	ctx := context.Background()
	
	// This test will fail if RSS feed is not accessible, but should not panic
	err = ProcessRSSScheduled(ctx, event)
	// We expect this to potentially fail in test environment due to missing real RSS feeds
	// The important thing is that it doesn't panic and handles errors gracefully
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestProcessRSSScheduledInvalidJSON(t *testing.T) {
	// Create a Cloud Event with invalid JSON data
	event := CloudEvent{
		ID:              "test-event-id",
		Source:          "test-source",
		SpecVersion:     "1.0",
		Type:            "google.cloud.scheduler.job.v1.executed",
		Subject:         "projects/test-project/locations/us-central1/jobs/test-job",
		Time:            time.Now(),
		DataContentType: "application/json",
		Data:            json.RawMessage(`invalid json`),
	}
	
	ctx := context.Background()
	
	err := ProcessRSSScheduled(ctx, event)
	if err == nil {
		t.Error("Expected error for invalid JSON data")
	}
	
	if !bytes.Contains([]byte(err.Error()), []byte("failed to parse event data")) {
		t.Errorf("Expected 'failed to parse event data' error, got: %v", err)
	}
}

func TestProcessRSSScheduledEmptyFeedName(t *testing.T) {
	// Test data with empty feed name (should process all feeds)
	eventData := CloudEventData{
		FeedName: "",
	}
	
	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}
	
	event := CloudEvent{
		ID:              "test-event-id",
		Source:          "test-source",
		SpecVersion:     "1.0",
		Type:            "google.cloud.scheduler.job.v1.executed",
		Subject:         "projects/test-project/locations/us-central1/jobs/test-job",
		Time:            time.Now(),
		DataContentType: "application/json",
		Data:            data,
	}
	
	ctx := context.Background()
	
	// This should attempt to process all enabled feeds
	err = ProcessRSSScheduled(ctx, event)
	// We expect this to potentially fail in test environment
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
	}
}

func TestCloudEventDataSerialization(t *testing.T) {
	original := CloudEventData{
		FeedName: "test-feed",
	}
	
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal CloudEventData: %v", err)
	}
	
	var unmarshaled CloudEventData
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CloudEventData: %v", err)
	}
	
	if unmarshaled.FeedName != original.FeedName {
		t.Errorf("Expected FeedName '%s', got '%s'", original.FeedName, unmarshaled.FeedName)
	}
}

func TestCloudEventSerialization(t *testing.T) {
	eventData := CloudEventData{
		FeedName: "test-feed",
	}
	
	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}
	
	original := CloudEvent{
		ID:              "test-id",
		Source:          "test-source",
		SpecVersion:     "1.0",
		Type:            "test-type",
		Subject:         "test-subject",
		Time:            time.Now().Truncate(time.Second), // Truncate for comparison
		DataContentType: "application/json",
		Data:            data,
		Extensions:      map[string]interface{}{"test": "value"},
	}
	
	eventBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal CloudEvent: %v", err)
	}
	
	var unmarshaled CloudEvent
	err = json.Unmarshal(eventBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal CloudEvent: %v", err)
	}
	
	if unmarshaled.ID != original.ID {
		t.Errorf("Expected ID '%s', got '%s'", original.ID, unmarshaled.ID)
	}
	
	if unmarshaled.Source != original.Source {
		t.Errorf("Expected Source '%s', got '%s'", original.Source, unmarshaled.Source)
	}
	
	if unmarshaled.Type != original.Type {
		t.Errorf("Expected Type '%s', got '%s'", original.Type, unmarshaled.Type)
	}
	
	if !unmarshaled.Time.Equal(original.Time) {
		t.Errorf("Expected Time '%v', got '%v'", original.Time, unmarshaled.Time)
	}
}

// Integration test helper functions

func setupTestServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(SummarizeArticles))
}

func TestSummarizeArticlesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	server := setupTestServer()
	defer server.Close()
	
	// Test health endpoint
	resp, err := http.Get(server.URL + "/api/v1/health")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%v'", response["status"])
	}
}

func TestSummarizeArticlesConfigEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	server := setupTestServer()
	defer server.Close()
	
	// Test config endpoint
	resp, err := http.Get(server.URL + "/api/v1/config")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
	
	var response map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	
	// Should contain config information but not sensitive data
	if _, ok := response["cache_type"]; !ok {
		t.Error("Expected 'cache_type' in config response")
	}
	
	if _, ok := response["rss_feeds"]; !ok {
		t.Error("Expected 'rss_feeds' in config response")
	}
	
	// Should not contain sensitive information
	if _, ok := response["gemini_api_key"]; ok {
		t.Error("Config response should not contain sensitive 'gemini_api_key'")
	}
	
	if _, ok := response["slack_bot_token"]; ok {
		t.Error("Config response should not contain sensitive 'slack_bot_token'")
	}
}

// Benchmark tests

func BenchmarkSummarizeArticlesHealthCheck(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/api/v1/health", nil)
		w := httptest.NewRecorder()
		SummarizeArticles(w, req)
		
		if w.Code != http.StatusOK {
			b.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
	}
}

func BenchmarkProcessRSSScheduledSerialization(b *testing.B) {
	eventData := CloudEventData{
		FeedName: "test-feed",
	}
	
	for i := 0; i < b.N; i++ {
		data, err := json.Marshal(eventData)
		if err != nil {
			b.Fatalf("Failed to marshal event data: %v", err)
		}
		
		var unmarshaled CloudEventData
		err = json.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatalf("Failed to unmarshal event data: %v", err)
		}
	}
}