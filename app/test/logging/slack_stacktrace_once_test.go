package logging

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/application"
)

// TestSlackStackTrace_OnceOnly verifies Slack errors produce exactly one stack trace from slack.go.
func TestSlackStackTrace_OnceOnly(t *testing.T) {
	// Create mock server for Gemini (success)
	geminiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{
			"candidates": [{
				"content": {
					"parts": [{"text": "Mock summary of the article"}]
				}
			}]
		}`)
	}))
	defer geminiServer.Close()

	// Create mock server for Slack (401 error)
	slackServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"error": "invalid_auth"}`)
	}))
	defer slackServer.Close()

	// Set up environment with mock URLs and unique bucket
	testID := fmt.Sprintf("%d", time.Now().UnixNano())
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("SLACK_BASE_URL", slackServer.URL)
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

	tests := []struct {
		name        string
		handler     http.Handler
		requestBody string
		endpoint    string
		description string
	}{
		{
			name:        "process_slack_error",
			handler:     app.HatenaHandler,
			requestBody: ``,
			endpoint:    "/process/hatena",
			description: "Hatena process flow Slack error should produce stack trace from slack.go",
		},
		{
			name:        "webhook_slack_error",
			handler:     app.WebhookHandler,
			requestBody: `{"url": "https://example.com"}`,
			endpoint:    "/webhook",
			description: "Webhook flow Slack error should produce stack trace from slack.go",
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

			// 1. Slackエラー直後のスタックトレースを確認
			slackStackPattern := regexp.MustCompile(`Slack API request failed[^\n]*\nStack:`)
			slackStackMatches := slackStackPattern.FindAllString(output, -1)

			// 3. 全体のスタックトレース数が1回のみ
			totalStacks := strings.Count(output, "Stack:")

			// 検証: Slackエラー直後のStackが1回 && 全体のStackが1回
			if len(slackStackMatches) != 1 {
				t.Errorf("%s: Expected exactly 1 Slack error followed by stack trace, got %d. Matches: %v",
					tt.description, len(slackStackMatches), slackStackMatches)
			} else {
				t.Logf("✅ %s: Found exactly 1 Slack error followed by stack trace", tt.description)
			}

			if totalStacks != 1 {
				t.Errorf("%s: Expected exactly 1 total stack trace, got %d. Output: %s",
					tt.description, totalStacks, output)
			} else {
				t.Logf("✅ %s: Found exactly 1 total stack trace (no other errors)", tt.description)
			}

			// Verify stack trace comes from slack.go (additional confirmation)
			if strings.Contains(output, "slack.go:") {
				t.Logf("✅ Stack trace contains slack.go - Slack repository error correctly traced")
			} else {
				t.Errorf("❌ Stack trace does not contain slack.go - not reaching Slack repository correctly")
				t.Logf("Full output: %s", output)
			}

			// Verify Slack error message is present
			if strings.Contains(output, "Slack API request failed") {
				t.Logf("✅ Found Slack API error message")
			} else {
				t.Logf("Note: Slack error message pattern may be in different format")
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
