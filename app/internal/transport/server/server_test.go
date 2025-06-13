package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// TestHealthCheck tests the health check endpoint directly
func TestHealthCheck(t *testing.T) {
	req := httptest.NewRequest("GET", "/hc", nil)
	w := httptest.NewRecorder()

	healthCheck(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", resp.Header.Get("Content-Type"))
	}

	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}
}

// TestCreateHandler tests handler creation with valid environment
func TestCreateHandler(t *testing.T) {
	// Set up minimal valid environment
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
	}()

	handler, cleanup, err := CreateHandler()
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	defer cleanup()

	if handler == nil {
		t.Error("Handler should not be nil")
	}

	// Test that the handler can handle health check requests without auth
	req := httptest.NewRequest("GET", "/hc", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	t.Logf("Health check: %s %s -> %d", req.Method, req.URL.Path, w.Code)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Test that webhook requires auth
	webhookReq := httptest.NewRequest("POST", "/webhook", nil)
	webhookW := httptest.NewRecorder()

	handler.ServeHTTP(webhookW, webhookReq)

	t.Logf("Webhook without auth: %s %s -> %d", webhookReq.Method, webhookReq.URL.Path, webhookW.Code)

	if webhookW.Code == http.StatusOK {
		t.Errorf("Webhook should require auth, but got status %d", webhookW.Code)
	}
}

// TestCreateHandler_InvalidEnv tests handler creation with invalid environment
func TestCreateHandler_InvalidEnv(t *testing.T) {
	// Clear environment variables
	originalGemini := os.Getenv("GEMINI_API_KEY")
	originalSlack := os.Getenv("SLACK_BOT_TOKEN")
	originalChannel := os.Getenv("SLACK_CHANNEL")

	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("SLACK_BOT_TOKEN")
	os.Unsetenv("SLACK_CHANNEL")

	defer func() {
		if originalGemini != "" {
			os.Setenv("GEMINI_API_KEY", originalGemini)
		}
		if originalSlack != "" {
			os.Setenv("SLACK_BOT_TOKEN", originalSlack)
		}
		if originalChannel != "" {
			os.Setenv("SLACK_CHANNEL", originalChannel)
		}
	}()

	_, _, err := CreateHandler()
	if err == nil {
		t.Error("Expected CreateHandler to fail with invalid environment, but it succeeded")
	}
}

// TestHandleRequest tests the Cloud Functions entry point
func TestHandleRequest(t *testing.T) {
	// Set up valid environment
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
	}()

	// Test health check through HandleRequest
	req := httptest.NewRequest("GET", "/hc", nil)
	w := httptest.NewRecorder()

	HandleRequest(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var result map[string]string
	if err := json.NewDecoder(w.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result["status"])
	}
}

// TestHandleRequest_InvalidEnv tests HandleRequest with invalid environment
func TestHandleRequest_InvalidEnv(t *testing.T) {
	// Clear environment variables
	originalGemini := os.Getenv("GEMINI_API_KEY")
	originalSlack := os.Getenv("SLACK_BOT_TOKEN")
	originalChannel := os.Getenv("SLACK_CHANNEL")

	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("SLACK_BOT_TOKEN")
	os.Unsetenv("SLACK_CHANNEL")

	defer func() {
		if originalGemini != "" {
			os.Setenv("GEMINI_API_KEY", originalGemini)
		}
		if originalSlack != "" {
			os.Setenv("SLACK_BOT_TOKEN", originalSlack)
		}
		if originalChannel != "" {
			os.Setenv("SLACK_CHANNEL", originalChannel)
		}
	}()

	req := httptest.NewRequest("GET", "/hc", nil)
	w := httptest.NewRecorder()

	HandleRequest(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", w.Code)
	}
}
