package integration

import (
	"os"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/application"
)

// 個別のテストで環境変数を設定するため、TestMainを削除

func TestIntegration_FullPipeline_MockedServices(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Ensure test environment is set up properly
	if os.Getenv("GEMINI_API_KEY") == "" {
		os.Setenv("GEMINI_API_KEY", "test-gemini-key")
		defer os.Unsetenv("GEMINI_API_KEY")
	}
	if os.Getenv("SLACK_BOT_TOKEN") == "" {
		os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
	}

	// Create application with real dependencies but test API keys
	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Test basic application creation succeeds
	if app.Config == nil {
		t.Error("Expected config to be loaded")
	}

	if app.HatenaHandler == nil {
		t.Error("Expected hatena handler to be created")
	}

	if app.WebhookHandler == nil {
		t.Error("Expected webhook handler to be created")
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
			_, err := application.Load()

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

	// Ensure test environment is set up properly
	if os.Getenv("GEMINI_API_KEY") == "" {
		os.Setenv("GEMINI_API_KEY", "test-gemini-key")
		defer os.Unsetenv("GEMINI_API_KEY")
	}
	if os.Getenv("SLACK_BOT_TOKEN") == "" {
		os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
	}

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Test application close
	err = app.Close()
	if err != nil {
		t.Fatalf("Failed to close application: %v", err)
	}
}

func TestIntegration_ApplicationLifecycle(t *testing.T) {
	// Ensure test environment is set up properly
	if os.Getenv("GEMINI_API_KEY") == "" {
		os.Setenv("GEMINI_API_KEY", "test-gemini-key")
		defer os.Unsetenv("GEMINI_API_KEY")
	}
	if os.Getenv("SLACK_BOT_TOKEN") == "" {
		os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
		defer os.Unsetenv("SLACK_BOT_TOKEN")
	}

	// Test application creation
	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}

	// Test application can be closed multiple times without error
	err = app.Close()
	if err != nil {
		t.Errorf("Failed to close application: %v", err)
	}

	err = app.Close()
	if err != nil {
		t.Errorf("Failed to close application second time: %v", err)
	}
}

func TestIntegration_FeedConfiguration(t *testing.T) {
	_, err := application.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Feed URLs are now configured in feed/config.go as default strategies
	// This test validates the basic config loading works
}

func TestIntegration_ErrorPropagation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	app, err := application.New()
	if err != nil {
		t.Fatalf("Failed to create application: %v", err)
	}
	defer app.Close()

	// Test that application creation succeeds even with test API keys
	// This validates that the dependency injection works correctly
	if app.Config.GeminiAPIKey != "test-gemini-key" {
		t.Errorf("Expected test API key, got: %s", app.Config.GeminiAPIKey)
	}

	if app.Config.SlackBotToken != "xoxb-test-token" {
		t.Errorf("Expected test Slack token, got: %s", app.Config.SlackBotToken)
	}

	t.Logf("Application created successfully with test configuration")
}

// Benchmark tests for integration scenarios.
func BenchmarkIntegration_ApplicationCreation(b *testing.B) {
	// Set test environment for benchmark
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	os.Setenv("SLACK_CHANNEL", "#test")
	defer func() {
		os.Unsetenv("GEMINI_API_KEY")
		os.Unsetenv("SLACK_BOT_TOKEN")
		os.Unsetenv("SLACK_CHANNEL")
	}()

	for i := 0; i < b.N; i++ {
		app, err := application.New()
		if err != nil {
			b.Fatalf("Failed to create application: %v", err)
		}
		app.Close()
	}
}

func BenchmarkIntegration_ConfigLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := application.Load()
		if err != nil {
			b.Fatalf("Failed to load config: %v", err)
		}
	}
}
