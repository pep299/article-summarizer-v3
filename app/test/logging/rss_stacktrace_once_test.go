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

	"github.com/pep299/article-summarizer-v3/internal/application"
)

// TestRSSStackTrace_OnceOnly verifies that RSS feed errors produce exactly one stack trace.
func TestRSSStackTrace_OnceOnly(t *testing.T) {
	// Create mock server for RSS (network error simulation)
	rssErrorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate network error by closing connection immediately
		hj, ok := w.(http.Hijacker)
		if ok {
			conn, _, _ := hj.Hijack()
			conn.Close()
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Network error")
	}))
	defer rssErrorServer.Close()

	// Create mock server for Gemini (success)
	geminiSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer geminiSuccessServer.Close()

	// Create mock server for Slack (success)
	slackSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer slackSuccessServer.Close()

	// Set up environment with mock servers
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiSuccessServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("SLACK_BASE_URL", slackSuccessServer.URL)
	os.Setenv("HATENA_RSS_URL", rssErrorServer.URL+"/feed.rss")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("SLACK_BASE_URL")
		os.Unsetenv("HATENA_RSS_URL")
	}()

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	tests := []struct {
		name                string
		handler             http.Handler
		requestBody         string
		endpoint            string
		expectedStackTraces int
		description         string
		errorPattern        string
	}{
		{
			name:                "rss_network_error",
			handler:             app.ProcessHandler,
			requestBody:         `{"feedName": "hatena"}`,
			endpoint:            "/",
			expectedStackTraces: 1, // RSS fetch network errorで1回だけ
			description:         "RSS feed network error should produce exactly 1 stack trace",
			errorPattern:        "Error making HTTP request to RSS feed",
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

			// 1. RSSエラー直後のスタックトレースを確認
			rssStackPattern := regexp.MustCompile(`Error making HTTP request to RSS feed[^\n]*\nStack:`)
			rssStackMatches := rssStackPattern.FindAllString(output, -1)

			// 3. 全体のスタックトレース数が1回のみ
			totalStacks := strings.Count(output, "Stack:")

			// 検証: RSSエラー直後のStackが1回 && 全体のStackが1回
			if len(rssStackMatches) != tt.expectedStackTraces {
				t.Errorf("%s: Expected exactly %d RSS error followed by stack trace, got %d. Matches: %v",
					tt.description, tt.expectedStackTraces, len(rssStackMatches), rssStackMatches)
			} else {
				t.Logf("✅ %s: Found exactly %d RSS error followed by stack trace", tt.description, len(rssStackMatches))
			}

			if totalStacks != tt.expectedStackTraces {
				t.Errorf("%s: Expected exactly %d total stack traces, got %d. Output: %s",
					tt.description, tt.expectedStackTraces, totalStacks, output)
			} else {
				t.Logf("✅ %s: Found exactly %d total stack trace(s) as expected", tt.description, totalStacks)
			}

			// Verify the stack trace appears with the correct error pattern
			if tt.expectedStackTraces > 0 {
				if !strings.Contains(output, tt.errorPattern) {
					t.Logf("Note: Expected error pattern '%s' may not be captured in stderr for %s",
						tt.errorPattern, tt.description)
				} else {
					t.Logf("✅ %s: Found expected error pattern '%s'", tt.description, tt.errorPattern)
				}
			}

			// Log the actual output for manual verification if needed
			if totalStacks > 0 {
				preview := output
				if len(output) > 200 {
					preview = output[:200]
				}
				t.Logf("Stack trace output preview for %s: %s...", tt.name,
					strings.ReplaceAll(preview, "\n", " "))
			}
		})
	}
}

// TestRSSStatusCodeError_OnceOnly verifies RSS status code errors produce exactly one stack trace.
func TestRSSStatusCodeError_OnceOnly(t *testing.T) {
	// Create mock server for RSS (404 error)
	rss404Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `<html><body><h1>404 Not Found</h1></body></html>`)
	}))
	defer rss404Server.Close()

	// Create mock server for Gemini (success)
	geminiSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer geminiSuccessServer.Close()

	// Create mock server for Slack (success)
	slackSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, `{"ok": true}`)
	}))
	defer slackSuccessServer.Close()

	// Set up environment with mock servers
	os.Setenv("GEMINI_API_KEY", "mock-key")
	os.Setenv("GEMINI_BASE_URL", geminiSuccessServer.URL)
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-mock-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("SLACK_BASE_URL", slackSuccessServer.URL)
	os.Setenv("HATENA_RSS_URL", rss404Server.URL+"/feed.rss")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("GEMINI_BASE_URL")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("SLACK_BASE_URL")
		os.Unsetenv("HATENA_RSS_URL")
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

	// 1. RSS 404エラー直後のスタックトレースを確認
	rss404StackPattern := regexp.MustCompile(`RSS feed request failed[^\n]*status_code=404[^\n]*\nStack:`)
	rss404StackMatches := rss404StackPattern.FindAllString(output, -1)

	// 3. 全体のスタックトレース数が1回のみ
	totalStacks := strings.Count(output, "Stack:")

	// 検証: RSS 404エラー直後のStackが1回 && 全体のStackが1回
	if len(rss404StackMatches) != 1 {
		t.Errorf("Expected exactly 1 RSS 404 error followed by stack trace, got %d. Matches: %v",
			len(rss404StackMatches), rss404StackMatches)
	} else {
		t.Logf("✅ Found exactly 1 RSS 404 error followed by stack trace")
	}

	if totalStacks != 1 {
		t.Errorf("Expected exactly 1 total stack trace for RSS 404 error, got %d. Output: %s",
			totalStacks, output)
	} else {
		t.Logf("✅ Found exactly 1 stack trace for RSS 404 error")
	}

	// Verify the error was logged appropriately
	if strings.Contains(output, "RSS feed request failed") && strings.Contains(output, "status_code=404") {
		t.Logf("✅ RSS 404 error was properly logged")
	} else {
		t.Logf("Note: RSS 404 error logging pattern not found in stderr capture")
	}
}
