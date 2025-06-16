package handler

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/mocks"
)

func TestHatenaHandler_ServeHTTP_GetMethod(t *testing.T) {
	// Set required environment variables
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
	}()

	handler := NewHatenaHandler(
		&mocks.MockHatenaRSSRepo{},
		&mocks.MockGeminiRepo{},
		&mocks.MockSlackRepo{},
		&mocks.MockProcessedRepo{},
		&mocks.MockLimiter{},
	)

	req := httptest.NewRequest("GET", "/process/hatena", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should still process successfully (no method restriction)
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

func TestHatenaHandler_ServeHTTP_ValidRequest(t *testing.T) {
	// Set required environment variables
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	os.Setenv("HATENA_RSS_URL", "http://localhost/test") // Use mock URL
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
		os.Unsetenv("HATENA_RSS_URL")
	}()

	handler := NewHatenaHandler(
		&mocks.MockHatenaRSSRepo{},
		&mocks.MockGeminiRepo{},
		&mocks.MockSlackRepo{},
		&mocks.MockProcessedRepo{},
		&mocks.MockLimiter{},
	)

	req := httptest.NewRequest("POST", "/process/hatena", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}
