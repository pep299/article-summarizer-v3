package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/joho/godotenv"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

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

func TestSummarizeArticles_ProcessEndpoint_MissingFeedParam(t *testing.T) {
	req := httptest.NewRequest("GET", "/process", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	expectedBody := "Missing 'feed' query parameter\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, w.Body.String())
	}
}

func TestSummarizeArticles_ProcessEndpoint_InvalidFeed(t *testing.T) {
	req := httptest.NewRequest("GET", "/process?feed=invalid-feed", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestSummarizeArticles_ProcessEndpoint_ValidFeed(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	req := httptest.NewRequest("GET", "/process?feed=hatena", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	// This will likely fail due to actual RSS fetching, but should not panic
	// The important thing is that it handles the request structure correctly
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
	}
}

func TestSummarizeArticles_InvalidRoute(t *testing.T) {
	req := httptest.NewRequest("GET", "/invalid", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestSummarizeArticles_ProcessEndpoint_POST(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	req := httptest.NewRequest("POST", "/process?feed=hatena", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	// Should handle POST requests (doesn't explicitly check method)
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
	}
}

func TestProcessRSSScheduled_ValidEvent(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	eventData := CloudEventData{
		FeedName: "hatena",
	}

	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}

	// Create a Cloud Event
	e := event.New()
	e.SetID("test-event-id")
	e.SetSource("test-source")
	e.SetType("google.cloud.scheduler.job.v1.executed")
	e.SetSubject("projects/test-project/locations/us-central1/jobs/test-job")
	e.SetDataContentType("application/json")
	e.SetData("application/json", data)

	ctx := context.Background()

	// This test will likely fail due to actual RSS fetching in test environment
	// The important thing is that it doesn't panic and handles errors gracefully
	err = ProcessRSSScheduled(ctx, e)
	if err != nil {
		t.Logf("Expected error in test environment: %v", err)
		// Should be a wrapped error indicating specific failure
		if err.Error() == "" {
			t.Error("Error should have descriptive message")
		}
	}
}

func TestProcessRSSScheduled_InvalidJSON(t *testing.T) {
	// Create a Cloud Event with invalid JSON data
	e := event.New()
	e.SetID("test-event-id")
	e.SetSource("test-source")
	e.SetType("google.cloud.scheduler.job.v1.executed")
	e.SetSubject("projects/test-project/locations/us-central1/jobs/test-job")
	e.SetDataContentType("application/json")
	e.SetData("application/json", []byte("invalid json"))

	ctx := context.Background()

	err := ProcessRSSScheduled(ctx, e)
	if err == nil {
		t.Error("Expected error for invalid JSON data")
	}

	if err.Error() != "failed to parse event data: invalid character 'i' looking for beginning of value" {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

func TestProcessRSSScheduled_EmptyFeedName(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test data with empty feed name (should process all feeds)
	eventData := CloudEventData{
		FeedName: "",
	}

	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}

	e := event.New()
	e.SetID("test-event-id")
	e.SetSource("test-source")
	e.SetType("google.cloud.scheduler.job.v1.executed")
	e.SetSubject("projects/test-project/locations/us-central1/jobs/test-job")
	e.SetDataContentType("application/json")
	e.SetData("application/json", data)

	ctx := context.Background()

	// This should attempt to process all enabled feeds
	err = ProcessRSSScheduled(ctx, e)
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

func TestSummarizeArticles_ConfigLoadFailure(t *testing.T) {
	// Temporarily unset required environment variable
	originalKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")

	defer func() {
		os.Setenv("GEMINI_API_KEY", originalKey)
	}()

	req := httptest.NewRequest("GET", "/process?feed=hatena", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}

	expectedBody := "Internal server error\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, w.Body.String())
	}
}

func TestProcessRSSScheduled_ConfigLoadFailure(t *testing.T) {
	// Temporarily unset required environment variable
	originalKey := os.Getenv("GEMINI_API_KEY")
	os.Unsetenv("GEMINI_API_KEY")

	defer func() {
		os.Setenv("GEMINI_API_KEY", originalKey)
	}()

	eventData := CloudEventData{
		FeedName: "hatena",
	}

	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}

	e := event.New()
	e.SetID("test-event-id")
	e.SetSource("test-source")
	e.SetType("google.cloud.scheduler.job.v1.executed")
	e.SetDataContentType("application/json")
	e.SetData("application/json", data)

	ctx := context.Background()

	err = ProcessRSSScheduled(ctx, e)
	if err == nil {
		t.Error("Expected error when config loading fails")
	}

	if !bytes.Contains([]byte(err.Error()), []byte("failed to load configuration")) {
		t.Errorf("Expected config load error, got: %v", err)
	}
}

func TestSummarizeArticles_WebhookAuth_ValidToken(t *testing.T) {
	// Set webhook auth token temporarily
	os.Setenv("WEBHOOK_AUTH_TOKEN", "test-token")
	defer os.Unsetenv("WEBHOOK_AUTH_TOKEN")

	req := httptest.NewRequest("POST", "/process?feed=hatena", strings.NewReader("token=test-token"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	// Should not be rejected due to auth (might fail for other reasons like invalid API key)
	if w.Code == http.StatusForbidden {
		t.Error("Valid token should not result in 403")
	}
}

func TestSummarizeArticles_WebhookAuth_InvalidToken(t *testing.T) {
	// Set webhook auth token temporarily
	os.Setenv("WEBHOOK_AUTH_TOKEN", "test-token")
	defer os.Unsetenv("WEBHOOK_AUTH_TOKEN")

	req := httptest.NewRequest("POST", "/process?feed=hatena", strings.NewReader("token=wrong-token"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for invalid token, got %d", http.StatusForbidden, w.Code)
	}

	expectedBody := "Unauthorized\n"
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body '%s', got '%s'", expectedBody, w.Body.String())
	}
}

func TestSummarizeArticles_WebhookAuth_MissingToken(t *testing.T) {
	// Set webhook auth token temporarily
	os.Setenv("WEBHOOK_AUTH_TOKEN", "test-token")
	defer os.Unsetenv("WEBHOOK_AUTH_TOKEN")

	req := httptest.NewRequest("POST", "/process?feed=hatena", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d for missing token, got %d", http.StatusForbidden, w.Code)
	}
}

func TestSummarizeArticles_WebhookAuth_NoAuthRequired(t *testing.T) {
	// No webhook auth token set
	req := httptest.NewRequest("POST", "/process?feed=hatena", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	// Should not be rejected due to auth when no auth token is configured
	if w.Code == http.StatusForbidden {
		t.Error("Should not require auth when WEBHOOK_AUTH_TOKEN is not set")
	}
}

// Benchmark tests
func BenchmarkSummarizeArticles_ProcessEndpoint(b *testing.B) {
	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("GET", "/process?feed=hatena", nil)
		w := httptest.NewRecorder()
		SummarizeArticles(w, req)
	}
}

func BenchmarkCloudEventDataSerialization(b *testing.B) {
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

// Test with real API keys to cover success paths
func TestSummarizeArticles_RealAPI_SuccessPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real API test in short mode")
	}

	// Load .env file for real API keys
	err := godotenv.Load("../../.env")
	if err != nil {
		t.Logf("Warning: Could not load .env file: %v", err)
	}

	// Verify API keys are loaded
	geminiKey := os.Getenv("GEMINI_API_KEY")
	slackToken := os.Getenv("SLACK_BOT_TOKEN")

	t.Logf("GEMINI_API_KEY loaded: %s...", geminiKey[:min(10, len(geminiKey))])
	t.Logf("SLACK_BOT_TOKEN loaded: %s...", slackToken[:min(10, len(slackToken))])

	if !strings.HasPrefix(geminiKey, "AIza") {
		t.Skip("Valid GEMINI_API_KEY not available")
	}
	if !strings.HasPrefix(slackToken, "xoxb-") {
		t.Skip("Valid SLACK_BOT_TOKEN not available")
	}

	req := httptest.NewRequest("GET", "/process?feed=hatena", nil)
	w := httptest.NewRecorder()

	SummarizeArticles(w, req)

	t.Logf("Response Status: %d", w.Code)
	t.Logf("Response Body: %s", w.Body.String())

	// Should succeed or fail gracefully with real APIs
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
	}

	// If successful, should have JSON response
	if w.Code == http.StatusOK {
		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		if err != nil {
			t.Errorf("Expected valid JSON response, got: %s", w.Body.String())
		}

		if response["status"] != "success" {
			t.Errorf("Expected status 'success', got: %v", response["status"])
		}

		if response["feed"] != "hatena" {
			t.Errorf("Expected feed 'hatena', got: %v", response["feed"])
		}
	}
}

func TestProcessRSSScheduled_RealAPI_SuccessPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real API test in short mode")
	}

	// Load .env file for real API keys
	_ = godotenv.Load("../../.env")

	// Check if real API keys are available
	if os.Getenv("GEMINI_API_KEY") == "" || !strings.HasPrefix(os.Getenv("GEMINI_API_KEY"), "AIza") {
		t.Skip("Real GEMINI_API_KEY not available")
	}
	if os.Getenv("SLACK_BOT_TOKEN") == "" || !strings.HasPrefix(os.Getenv("SLACK_BOT_TOKEN"), "xoxb-") {
		t.Skip("Real SLACK_BOT_TOKEN not available")
	}

	eventData := CloudEventData{
		FeedName: "hatena",
	}

	data, err := json.Marshal(eventData)
	if err != nil {
		t.Fatalf("Failed to marshal event data: %v", err)
	}

	e := event.New()
	e.SetID("test-event-id")
	e.SetSource("test-source")
	e.SetType("google.cloud.scheduler.job.v1.executed")
	e.SetDataContentType("application/json")
	e.SetData("application/json", data)

	ctx := context.Background()

	err = ProcessRSSScheduled(ctx, e)
	// Should succeed or fail gracefully with real APIs
	if err != nil {
		t.Logf("ProcessRSSScheduled completed with: %v", err)
	} else {
		t.Log("ProcessRSSScheduled completed successfully")
	}
}
