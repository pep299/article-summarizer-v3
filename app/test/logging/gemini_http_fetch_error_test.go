package logging

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/application"
)

// TestGeminiHTTPFetchError_DetailedLogging verifies that HTTP fetch errors produce detailed JSON logs.
func TestGeminiHTTPFetchError_DetailedLogging(t *testing.T) {
	// Create mock server for article URL (403 error)
	articleErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("X-Custom-Header", "test-value")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Access Denied: Insufficient permissions to access this resource")
	}))
	defer articleErrorServer.Close()

	// Create mock server for Gemini (success to avoid Gemini errors)
	geminiSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"candidates": [{"content": {"parts": [{"text": "Mock summary"}]}}]}`)
	}))
	defer geminiSuccessServer.Close()

	// Create mock server for Slack (success to avoid Slack errors)
	slackSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer slackSuccessServer.Close()

	// Set up environment with mock servers and unique bucket
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiSuccessServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("SLACK_BASE_URL", slackSuccessServer.URL)
	os.Setenv("CACHE_BUCKET", "test-bucket-"+testID)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("SLACK_BASE_URL")
		os.Unsetenv("CACHE_BUCKET")
	}()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Test webhook with 403 error from article URL
	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, articleErrorServer.URL)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Capture stderr where our logs are written
	oldStderr := os.Stderr
	r, w2, _ := os.Pipe()
	os.Stderr = w2

	// Execute request
	app.WebhookHandler.ServeHTTP(w, req)

	// Close pipe and restore stderr
	w2.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Find the HTTP request failed log
	lines := strings.Split(output, "\n")
	var httpErrorLogLine string
	for _, line := range lines {
		if strings.Contains(line, "HTTP request failed") {
			httpErrorLogLine = line
			break
		}
	}

	if httpErrorLogLine == "" {
		t.Fatalf("HTTP request failed log line not found in output: %s", output)
	}

	// Verify required information is present in log line
	// Format: "HTTP request failed url=... status_code=... request_headers=... response_headers=... response_body=..."

	// Verify status code
	if !strings.Contains(httpErrorLogLine, "status_code=403") {
		t.Errorf("Expected status_code=403 in log line, got: %s", httpErrorLogLine)
	}

	// Verify URL contains mock server address
	if !strings.Contains(httpErrorLogLine, "127.0.0.1") {
		t.Errorf("Expected URL to contain mock server address, got: %s", httpErrorLogLine)
	}

	// Verify response body contains error message
	if !strings.Contains(httpErrorLogLine, "Access Denied") {
		t.Errorf("Expected response body to contain 'Access Denied', got: %s", httpErrorLogLine)
	}

	// Verify request headers contain User-Agent
	if !strings.Contains(httpErrorLogLine, "Article Summarizer Bot") {
		t.Errorf("Expected User-Agent to contain 'Article Summarizer Bot', got: %s", httpErrorLogLine)
	}

	// Verify response headers contain Content-Type
	if !strings.Contains(httpErrorLogLine, "Content-Type:[text/html]") {
		t.Errorf("Expected Content-Type to contain 'text/html', got: %s", httpErrorLogLine)
	}

	// Verify custom header is present
	if !strings.Contains(httpErrorLogLine, "X-Custom-Header:[test-value]") {
		t.Errorf("Expected X-Custom-Header to be present, got: %s", httpErrorLogLine)
	}

	// Verify stack trace is present in output
	if !strings.Contains(output, "Stack:") {
		t.Errorf("Expected stack trace to be present in output")
	}

	// Verify stack trace contains gemini.go
	if !strings.Contains(output, "gemini.go") {
		t.Errorf("Expected stack trace to contain 'gemini.go'")
	}

	t.Logf("✅ HTTP fetch error log contains all required fields with correct values")
	t.Logf("✅ Error details logged in readable format")
	t.Logf("✅ Stack trace properly included and points to gemini.go")
}

// TestGeminiHTTPFetchError_StackTraceOnce verifies that HTTP fetch errors produce exactly one stack trace.
func TestGeminiHTTPFetchError_StackTraceOnce(t *testing.T) {
	// Create mock server for article URL (404 error)
	articleErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Page not found")
	}))
	defer articleErrorServer.Close()

	// Create mock server for Gemini (success)
	geminiSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"candidates": [{"content": {"parts": [{"text": "Mock summary"}]}}]}`)
	}))
	defer geminiSuccessServer.Close()

	// Create mock server for Slack (success)
	slackSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer slackSuccessServer.Close()

	// Set up environment
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiSuccessServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("SLACK_BASE_URL", slackSuccessServer.URL)
	os.Setenv("CACHE_BUCKET", "test-bucket-"+testID)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("SLACK_BASE_URL")
		os.Unsetenv("CACHE_BUCKET")
	}()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest("POST", "/webhook", strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, articleErrorServer.URL)))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Capture stderr
	oldStderr := os.Stderr
	r, w2, _ := os.Pipe()
	os.Stderr = w2

	// Execute request
	app.WebhookHandler.ServeHTTP(w, req)

	// Close pipe and restore stderr
	w2.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Count stack traces - should be exactly 1
	stackTraceCount := strings.Count(output, "Stack:")
	if stackTraceCount != 1 {
		t.Errorf("Expected exactly 1 stack trace, got %d. Output: %s", stackTraceCount, output)
	} else {
		t.Logf("✅ Found exactly 1 stack trace for HTTP fetch error")
	}

	// Verify the request stopped due to HTTP error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500 (internal error), got %d", w.Code)
	} else {
		t.Logf("✅ Request correctly failed with HTTP fetch error")
	}

	// Verify HTTP request failed message is present
	if !strings.Contains(output, "HTTP request failed") {
		t.Errorf("Expected 'HTTP request failed' message in logs")
	} else {
		t.Logf("✅ HTTP request failed message properly logged")
	}
}

// TestGeminiHTTPFetchError_DifferentStatusCodes tests different HTTP error codes.
func TestGeminiHTTPFetchError_DifferentStatusCodes(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
		body       string
	}{
		{"403_forbidden", http.StatusForbidden, "Access forbidden"},
		{"404_not_found", http.StatusNotFound, "Page not found"},
		{"500_server_error", http.StatusInternalServerError, "Internal server error"},
		{"429_rate_limit", http.StatusTooManyRequests, "Rate limit exceeded"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock server for specific error
			errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
				fmt.Fprint(w, tc.body)
			}))
			defer errorServer.Close()

			// Set up environment
			testID := fmt.Sprintf("%d", time.Now().UnixNano())
			os.Setenv("GEMINI_API_KEY", "mock-key")
			os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
			os.Setenv("SLACK_CHANNEL", "#test")
			os.Setenv("CACHE_BUCKET", "test-bucket-"+testID)
			defer func() {
				os.Unsetenv("GEMINI_API_KEY")
				os.Unsetenv("SLACK_BOT_TOKEN")
				os.Unsetenv("SLACK_CHANNEL")
				os.Unsetenv("CACHE_BUCKET")
			}()

			app, err := application.New()
			if err != nil {
				t.Fatalf("Failed to create application: %v", err)
			}
			defer app.Close()

			req := httptest.NewRequest("POST", "/webhook", strings.NewReader(fmt.Sprintf(`{"url": "%s"}`, errorServer.URL)))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			// Capture stderr
			oldStderr := os.Stderr
			r, w2, _ := os.Pipe()
			os.Stderr = w2

			// Execute request
			app.WebhookHandler.ServeHTTP(w, req)

			// Close pipe and restore stderr
			w2.Close()
			os.Stderr = oldStderr

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// Find HTTP error log line
			lines := strings.Split(output, "\n")
			var httpErrorLogLine string
			for _, line := range lines {
				if strings.Contains(line, "HTTP request failed") {
					httpErrorLogLine = line
					break
				}
			}

			if httpErrorLogLine == "" {
				t.Fatalf("HTTP error log not found for status %d", tc.statusCode)
			}

			// Verify status code
			expectedStatusCode := fmt.Sprintf("status_code=%d", tc.statusCode)
			if !strings.Contains(httpErrorLogLine, expectedStatusCode) {
				t.Errorf("Expected status_code %d, got: %s", tc.statusCode, httpErrorLogLine)
			}

			// Verify response body
			if !strings.Contains(httpErrorLogLine, tc.body) {
				t.Errorf("Expected response body to contain '%s', got: %s", tc.body, httpErrorLogLine)
			}

			t.Logf("✅ Status %d properly logged with detailed error information", tc.statusCode)
		})
	}
}
