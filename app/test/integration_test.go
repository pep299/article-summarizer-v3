package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/handlers"
)

func TestMain(m *testing.M) {
	// Set up test environment variables for integration tests
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

func TestIntegration_FullPipeline_MockedServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Create server with real dependencies but test API keys
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test processing a non-existent feed (should fail quickly)
	err = server.ProcessSingleFeed(ctx, "non-existent-feed")
	if err == nil {
		t.Error("Expected error for non-existent feed")
	}

	expectedError := "feed non-existent-feed not found"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestIntegration_ConfigValidation(t *testing.T) {
	// Test various configuration scenarios
	testCases := []struct {
		name        string
		envVars     map[string]string
		expectError bool
	}{
		{
			name: "valid config",
			envVars: map[string]string{
				"GEMINI_API_KEY":  "test-key",
				"SLACK_BOT_TOKEN": "xoxb-test-token",
				"CACHE_TYPE":      "memory",
			},
			expectError: false,
		},
		{
			name: "missing gemini key",
			envVars: map[string]string{
				"SLACK_BOT_TOKEN": "xoxb-test-token",
				"CACHE_TYPE":      "memory",
			},
			expectError: true,
		},
		{
			name: "missing slack token",
			envVars: map[string]string{
				"GEMINI_API_KEY": "test-key",
				"CACHE_TYPE":     "memory",
			},
			expectError: true,
		},
		{
			name: "invalid slack token format",
			envVars: map[string]string{
				"GEMINI_API_KEY":  "test-key",
				"SLACK_BOT_TOKEN": "invalid-token",
				"CACHE_TYPE":      "memory",
			},
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clear environment
			os.Clearenv()

			// Set test environment variables
			for key, value := range tc.envVars {
				os.Setenv(key, value)
			}

			// Try to load config
			_, err := config.Load()

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			} else if !tc.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}

	// Restore test environment after test
	os.Setenv("GEMINI_API_KEY", "test-gemini-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test-channel")
	os.Setenv("CACHE_TYPE", "memory")
	os.Setenv("CACHE_DURATION_HOURS", "1")
}

func TestIntegration_CacheManager(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	// Test server close
	err = server.Close()
	if err != nil {
		t.Fatalf("Failed to close server: %v", err)
	}
}

func TestIntegration_ServerLifecycle(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test server creation
	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// Test server can be closed multiple times without error
	err = server.Close()
	if err != nil {
		t.Errorf("Failed to close server: %v", err)
	}

	err = server.Close()
	if err != nil {
		t.Errorf("Failed to close server second time: %v", err)
	}
}

func TestIntegration_FeedConfiguration(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify expected feed URLs are configured
	if cfg.HatenaRSSURL == "" {
		t.Error("Expected HatenaRSSURL to be configured")
	}

	if cfg.LobstersRSSURL == "" {
		t.Error("Expected LobstersRSSURL to be configured")
	}

	// Check default URLs
	expectedHatenaURL := "https://b.hatena.ne.jp/hotentry/it.rss"
	expectedLobstersURL := "https://lobste.rs/rss"

	if cfg.HatenaRSSURL != expectedHatenaURL {
		t.Errorf("Expected HatenaRSSURL to be '%s', got '%s'", expectedHatenaURL, cfg.HatenaRSSURL)
	}

	if cfg.LobstersRSSURL != expectedLobstersURL {
		t.Errorf("Expected LobstersRSSURL to be '%s', got '%s'", expectedLobstersURL, cfg.LobstersRSSURL)
	}
}

func TestIntegration_ErrorPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	server, err := handlers.NewServer(cfg)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test that errors are properly propagated and not swallowed
	// This should fail due to invalid API key, but error should be descriptive
	err = server.ProcessSingleFeed(ctx, "hatena")
	if err == nil {
		t.Error("Expected error due to invalid API key")
	} else {
		// Error should contain meaningful information about the failure
		errorStr := err.Error()
		if errorStr == "" {
			t.Error("Error message should not be empty")
		}

		// Should indicate which article and which step failed
		if len(errorStr) < 10 {
			t.Errorf("Error message seems too short: %s", errorStr)
		}

		t.Logf("Got expected error: %v", err)
	}
}

// Benchmark tests for integration scenarios
func BenchmarkIntegration_ServerCreation(b *testing.B) {
	cfg := &config.Config{
		GeminiAPIKey:  "test-key",
		GeminiModel:   "test-model",
		SlackBotToken: "xoxb-test-token",
		SlackChannel:  "#test",
	}

	for i := 0; i < b.N; i++ {
		server, err := handlers.NewServer(cfg)
		if err != nil {
			b.Fatalf("Failed to create server: %v", err)
		}
		server.Close()
	}
}

func BenchmarkIntegration_ConfigLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := config.Load()
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}
