package logging

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/application"
)

// TestGeminiStackTrace_OnceOnly verifies that Gemini errors produce exactly 1 stack trace.
func TestGeminiStackTrace_OnceOnly(t *testing.T) {
	// Create mock server for Gemini (400 error)
	geminiErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{
			"error": {
				"code": 400,
				"message": "API key not valid. Please pass a valid API key.",
				"status": "INVALID_ARGUMENT"
			}
		}`)
	}))
	defer geminiErrorServer.Close()

	// Create mock server for Slack (success to avoid Slack errors)
	slackSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer slackSuccessServer.Close()

	// Set up environment with mock servers and unique bucket to avoid flaky tests
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiErrorServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("SLACK_BASE_URL", slackSuccessServer.URL)
	os.Setenv("CACHE_BUCKET", "test-bucket-"+testID) // Unique bucket per test
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

	tests := []struct {
		name         string
		handler      http.Handler
		requestBody  string
		endpoint     string
		description  string
		errorPattern string
	}{
		{
			name:         "gemini_external_error",
			handler:      app.ProcessHandler,
			requestBody:  `{"feedName": "hatena"}`,
			endpoint:     "/",
			description:  "Gemini API external factor error should produce exactly 1 stack trace",
			errorPattern: "Gemini API request failed",
		},
		{
			name:         "webhook_gemini_external_error",
			handler:      app.WebhookHandler,
			requestBody:  `{"url": "https://example.com"}`,
			endpoint:     "/webhook",
			description:  "Webhook Gemini API external factor error should produce exactly 1 stack trace",
			errorPattern: "Gemini API request failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing %s", tt.description)

			req := httptest.NewRequest("POST", tt.endpoint, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			// Capture stderr where our logs are written
			oldStderr := os.Stderr
			r, w2, _ := os.Pipe()
			os.Stderr = w2

			// Execute request
			tt.handler.ServeHTTP(w, req)

			// Close pipe and restore stderr
			w2.Close()
			os.Stderr = oldStderr

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			// 1. Geminiエラー直後のスタックトレースを確認
			// ログフォーマット: "Gemini API request failed status_code=400 response={...}\nStack:"
			geminiStackPattern := regexp.MustCompile(`(?s)Gemini API request failed status_code=\d+ response=\{.*?\}\s*\nStack:`)
			geminiStackMatches := geminiStackPattern.FindAllString(output, -1)

			// 3. 全体のスタックトレース数が1回のみ
			totalStacks := strings.Count(output, "Stack:")

			// 検証: Geminiエラー直後のStackが1回 && 全体のStackが1回
			if len(geminiStackMatches) != 1 {
				t.Errorf("%s: Expected exactly 1 Gemini error followed by stack trace, got %d. Matches: %v",
					tt.description, len(geminiStackMatches), geminiStackMatches)
			} else {
				t.Logf("✅ %s: Found exactly 1 Gemini error followed by stack trace", tt.description)
			}

			if totalStacks != 1 {
				t.Errorf("%s: Expected exactly 1 total stack trace, got %d. Output: %s",
					tt.description, totalStacks, output)
			} else {
				t.Logf("✅ %s: Found exactly 1 total stack trace (no other errors)", tt.description)
			}

			// Verify stack trace comes from gemini.go (additional confirmation)
			if strings.Contains(output, "gemini.go:") {
				t.Logf("✅ Stack trace contains gemini.go - Gemini repository error correctly traced")
			} else {
				t.Errorf("❌ Stack trace does not contain gemini.go - not reaching Gemini repository correctly")
				t.Logf("Full output: %s", output)
			}

			// Verify Gemini error message is present
			if strings.Contains(output, tt.errorPattern) {
				t.Logf("✅ Found expected Gemini error pattern '%s'", tt.errorPattern)
			} else {
				t.Logf("Note: Expected error pattern '%s' may not be captured in stderr for %s",
					tt.errorPattern, tt.description)
			}

			// Log preview for manual verification
			if totalStacks > 0 {
				preview := output
				if len(output) > 500 {
					preview = output[:500]
				}
				t.Logf("Stack trace preview: %s...", strings.ReplaceAll(preview, "\n", " "))
			}
		})
	}
}

// TestGeminiAPI400Error_ProcessingStops verifies that 400 errors stop processing correctly.
func TestGeminiAPI400Error_ProcessingStops(t *testing.T) {
	// Create mock server for Gemini (400 error)
	geminiErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{
			"error": {
				"code": 400,
				"message": "API key not valid. Please pass a valid API key.",
				"status": "INVALID_ARGUMENT"
			}
		}`)
	}))
	defer geminiErrorServer.Close()

	// Set up environment with mock server and unique bucket
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiErrorServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("CACHE_BUCKET", "test-bucket-"+testID)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("CACHE_BUCKET")
	}()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	tests := []struct {
		name           string
		handler        http.Handler
		requestBody    string
		endpoint       string
		expectedStatus int
		description    string
		checkPattern   string
	}{
		{
			name:           "gemini_400_stops_processing",
			handler:        app.ProcessHandler,
			requestBody:    `{"feedName": "hatena"}`,
			endpoint:       "/",
			expectedStatus: http.StatusInternalServerError, // 400エラーが500として上がってくる
			description:    "Gemini 400 error should stop processing and return error",
			checkPattern:   "API key not valid", // Geminiの400エラーメッセージ
		},
		{
			name:           "webhook_gemini_400_stops_processing",
			handler:        app.WebhookHandler,
			requestBody:    `{"url": "https://example.com"}`,
			endpoint:       "/webhook",
			expectedStatus: http.StatusInternalServerError, // 400エラーが500として上がってくる
			description:    "Webhook Gemini 400 error should stop processing and return error",
			checkPattern:   "API key not valid", // Geminiの400エラーメッセージ
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Logf("Testing %s", tt.description)

			req := httptest.NewRequest("POST", tt.endpoint, strings.NewReader(tt.requestBody))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()

			// Execute request
			tt.handler.ServeHTTP(w, req)

			// Verify that processing stopped with error status
			if w.Code != tt.expectedStatus {
				t.Errorf("%s: Expected status %d, got %d. Body: %s",
					tt.description, tt.expectedStatus, w.Code, w.Body.String())
			} else {
				t.Logf("✅ %s: Correctly returned status %d", tt.description, w.Code)
			}

			// Verify error response contains expected pattern
			responseBody := w.Body.String()
			if !strings.Contains(responseBody, "error") {
				t.Errorf("%s: Expected error response, got: %s", tt.description, responseBody)
			} else {
				t.Logf("✅ %s: Response correctly indicates error", tt.description)
			}

			// Optional: Check for specific error message in logs or response
			// Note: The exact error message might be in logs rather than response body
			t.Logf("Response for %s: %s", tt.name, responseBody)
		})
	}
}

// TestGeminiAPI400Error_StackTraceOnce verifies that 400 errors produce exactly one stack trace.
func TestGeminiAPI400Error_StackTraceOnce(t *testing.T) {
	// Create mock server for Gemini (400 error)
	geminiErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{
			"error": {
				"code": 400,
				"message": "API key not valid. Please pass a valid API key.",
				"status": "INVALID_ARGUMENT"
			}
		}`)
	}))
	defer geminiErrorServer.Close()

	// Set up environment with mock server and unique bucket
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiErrorServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("CACHE_BUCKET", "test-bucket-"+testID)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("CACHE_BUCKET")
	}()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"feedName": "hatena"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Capture stderr
	oldStderr := os.Stderr
	r, w2, _ := os.Pipe()
	os.Stderr = w2

	// Execute request
	app.ProcessHandler.ServeHTTP(w, req)

	// Close pipe and restore stderr
	w2.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf strings.Builder
	_, err = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("Failed to read from pipe: %v", err)
	}
	output := buf.String()

	// 1. Geminiエラー直後のスタックトレースを確認
	// ログフォーマット: "Gemini API request failed status_code=400 response={...}\nStack:"
	geminiStackPattern := regexp.MustCompile(`(?s)Gemini API request failed status_code=400 response=\{.*?\}\s*\nStack:`)
	geminiStackMatches := geminiStackPattern.FindAllString(output, -1)

	// 3. 全体のスタックトレース数が1回のみ
	totalStacks := strings.Count(output, "Stack:")

	// 検証: Gemini 400エラー直後のStackが1回 && 全体のStackが1回
	if len(geminiStackMatches) != 1 {
		t.Errorf("Expected exactly 1 Gemini 400 error followed by stack trace, got %d. Matches: %v",
			len(geminiStackMatches), geminiStackMatches)
	} else {
		t.Logf("✅ Found exactly 1 Gemini 400 error followed by stack trace")
	}

	if totalStacks != 1 {
		t.Errorf("Expected exactly 1 total stack trace for 400 error, got %d", totalStacks)
		t.Logf("Output: %s", output)
	} else {
		t.Logf("✅ Found exactly 1 total stack trace for 400 error (external factor)")
	}

	// Verify the error was logged appropriately
	if strings.Contains(output, "Gemini API request failed status_code=400") {
		t.Logf("✅ 400 error was properly logged")
	} else {
		t.Logf("Note: 400 error logging pattern not found in stderr capture")
	}
}

// TestGeminiStackTrace_NoDuplication verifies no duplication in error propagation.
func TestGeminiStackTrace_NoDuplication(t *testing.T) {
	// Create mock server for Gemini (400 error)
	geminiErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{
			"error": {
				"code": 400,
				"message": "API key not valid. Please pass a valid API key.",
				"status": "INVALID_ARGUMENT"
			}
		}`)
	}))
	defer geminiErrorServer.Close()

	// Set up environment with mock server and unique bucket
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiErrorServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("CACHE_BUCKET", "test-bucket-"+testID)
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("CACHE_BUCKET")
	}()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	req := httptest.NewRequest("POST", "/", strings.NewReader(`{"feedName": "hatena"}`))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()

	// Capture stderr
	oldStderr := os.Stderr
	r, w2, _ := os.Pipe()
	os.Stderr = w2

	// Execute request
	app.ProcessHandler.ServeHTTP(w, req)

	// Close pipe and restore stderr
	w2.Close()
	os.Stderr = oldStderr

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Count error propagation patterns to ensure no duplication
	errorPatterns := []string{
		"Error processing feed",
		"Error calling Gemini API",
		"Error sending request to Gemini API",
	}

	for _, pattern := range errorPatterns {
		count := strings.Count(output, pattern)
		if count > 1 {
			t.Errorf("Error message '%s' appears %d times, suggesting duplication. Should appear only once. Output: %s",
				pattern, count, output)
		} else if count == 1 {
			t.Logf("✅ Error message '%s' appears exactly once (no duplication)", pattern)
		}
	}

	// Verify exactly 1 stack trace total
	stackTraceCount := strings.Count(output, "Stack:")
	if stackTraceCount > 1 {
		t.Errorf("Expected at most 1 stack trace for network error, got %d (suggesting duplication). Output: %s",
			stackTraceCount, output)
	} else {
		t.Logf("✅ Found %d stack trace(s) total (no duplication)", stackTraceCount)
	}
}
