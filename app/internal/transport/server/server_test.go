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

// TestHTTPMethodRouting tests Go 1.22 method-specific routing
func TestHTTPMethodRouting(t *testing.T) {
	// Set up valid environment
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("WEBHOOK_AUTH_TOKEN", "test-token")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("WEBHOOK_AUTH_TOKEN")
	}()

	handler, cleanup, err := CreateHandler()
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}
	defer cleanup()

	tests := []struct {
		name           string
		method         string
		path           string
		auth           bool
		expectedStatus int
		description    string
	}{
		{"Health Check GET", "GET", "/hc", false, 200, "Health check should work without auth"},
		{"Process GET", "GET", "/process", false, 405, "GET /process should return 405"},
		{"Process POST no auth", "POST", "/process", false, 401, "POST /process without auth should return 401"},
		{"Process POST with auth", "POST", "/process", true, 400, "POST /process with auth should return 400 (invalid JSON)"},
		{"Process sub-path GET", "GET", "/process/1", false, 404, "GET /process/1 should return 404"},
		{"Process sub-path POST", "POST", "/process/1", false, 404, "POST /process/1 should return 404"},
		{"Process DELETE", "DELETE", "/process", true, 405, "DELETE /process should return 405"},
		{"Webhook POST with auth", "POST", "/webhook", true, 400, "POST /webhook should work with auth"},
		{"Webhook GET", "GET", "/webhook", true, 405, "GET /webhook should return 405"},
		{"X GET no auth", "GET", "/x", false, 401, "GET /x without auth should return 401"},
		{"X POST with auth", "POST", "/x", true, 405, "POST /x should return 405"},
		{"X GET with auth but no URL", "GET", "/x", true, 400, "GET /x with auth but no URL should return 400"},
		{"X GET with auth and invalid URL", "GET", "/x?url=https://invalid.com", true, 400, "GET /x with invalid URL should return 400"},
		{"X quote chain GET no auth", "GET", "/x/quote-chain", false, 401, "GET /x/quote-chain without auth should return 401"},
		{"X quote chain POST with auth", "POST", "/x/quote-chain", true, 405, "POST /x/quote-chain should return 405"},
		{"X quote chain GET with auth but no URL", "GET", "/x/quote-chain", true, 400, "GET /x/quote-chain with auth but no URL should return 400"},
		{"X quote chain GET with auth and invalid URL", "GET", "/x/quote-chain?url=https://invalid.com", true, 400, "GET /x/quote-chain with invalid URL should return 400"},
		{"Root GET", "GET", "/", false, 404, "GET / should return 404"},
		{"Root POST", "POST", "/", false, 404, "POST / should return 404"},
		{"Unknown GET", "GET", "/unknown", false, 404, "GET /unknown should return 404"},
		{"Unknown POST", "POST", "/unknown", true, 404, "POST /unknown should return 404"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.auth {
				req.Header.Set("Authorization", "Bearer test-token")
			}
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			t.Logf("%s %s (auth: %v) -> %d", tt.method, tt.path, tt.auth, w.Code)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d - %s", tt.expectedStatus, w.Code, tt.description)
			}
		})
	}
}
