package infrastructure

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Set some test environment variables
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
	defer os.Unsetenv("GEMINI_API_KEY")
	defer os.Unsetenv("SLACK_BOT_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GeminiAPIKey != "test-key" {
		t.Errorf("Expected GeminiAPIKey to be 'test-key', got '%s'", cfg.GeminiAPIKey)
	}

	if cfg.SlackBotToken != "xoxb-test-token" {
		t.Errorf("Expected SlackBotToken to be 'xoxb-test-token', got '%s'", cfg.SlackBotToken)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected Port to be '8080', got '%s'", cfg.Port)
	}

	if cfg.GeminiModel != "gemini-2.5-flash-preview-05-20" {
		t.Errorf("Expected GeminiModel to be 'gemini-2.5-flash-preview-05-20', got '%s'", cfg.GeminiModel)
	}

	if cfg.SlackChannel != "#dev-null" {
		t.Errorf("Expected SlackChannel to be '#dev-null', got '%s'", cfg.SlackChannel)
	}

	if cfg.WebhookSlackChannel != "#dev-null" {
		t.Errorf("Expected WebhookSlackChannel to be '#dev-null', got '%s'", cfg.WebhookSlackChannel)
	}

	// Check RSS feeds configuration
	if cfg.HatenaRSSURL == "" {
		t.Error("Expected HatenaRSSURL to be configured")
	}

	if cfg.LobstersRSSURL == "" {
		t.Error("Expected LobstersRSSURL to be configured")
	}

	expectedHatenaURL := "https://b.hatena.ne.jp/hotentry/it.rss"
	if cfg.HatenaRSSURL != expectedHatenaURL {
		t.Errorf("Expected HatenaRSSURL to be '%s', got '%s'", expectedHatenaURL, cfg.HatenaRSSURL)
	}

	expectedLobstersURL := "https://lobste.rs/rss"
	if cfg.LobstersRSSURL != expectedLobstersURL {
		t.Errorf("Expected LobstersRSSURL to be '%s', got '%s'", expectedLobstersURL, cfg.LobstersRSSURL)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupEnv    func()
		cleanupEnv  func()
		expectError bool
		errorField  string
	}{
		{
			name: "missing GEMINI_API_KEY",
			setupEnv: func() {
				os.Unsetenv("GEMINI_API_KEY")
				os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
			},
			cleanupEnv: func() {
				os.Unsetenv("SLACK_BOT_TOKEN")
			},
			expectError: true,
			errorField:  "GEMINI_API_KEY",
		},
		{
			name: "missing SLACK_BOT_TOKEN",
			setupEnv: func() {
				os.Setenv("GEMINI_API_KEY", "test-key")
				os.Unsetenv("SLACK_BOT_TOKEN")
			},
			cleanupEnv: func() {
				os.Unsetenv("GEMINI_API_KEY")
			},
			expectError: true,
			errorField:  "SLACK_BOT_TOKEN",
		},
		{
			name: "invalid SLACK_BOT_TOKEN prefix",
			setupEnv: func() {
				os.Setenv("GEMINI_API_KEY", "test-key")
				os.Setenv("SLACK_BOT_TOKEN", "invalid-token")
			},
			cleanupEnv: func() {
				os.Unsetenv("GEMINI_API_KEY")
				os.Unsetenv("SLACK_BOT_TOKEN")
			},
			expectError: true,
			errorField:  "SLACK_BOT_TOKEN",
		},
		{
			name: "valid configuration",
			setupEnv: func() {
				os.Setenv("GEMINI_API_KEY", "test-key")
				os.Setenv("SLACK_BOT_TOKEN", "xoxb-test-token")
			},
			cleanupEnv: func() {
				os.Unsetenv("GEMINI_API_KEY")
				os.Unsetenv("SLACK_BOT_TOKEN")
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.setupEnv()
			defer test.cleanupEnv()

			_, err := Load()
			if test.expectError && err == nil {
				t.Errorf("Expected validation error for %s", test.errorField)
			}
			if !test.expectError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
			if test.expectError && err != nil {
				configErr, ok := err.(*ConfigError)
				if !ok {
					t.Errorf("Expected ConfigError, got %T", err)
				} else if configErr.Field != test.errorField {
					t.Errorf("Expected error field '%s', got '%s'", test.errorField, configErr.Field)
				}
			}
		})
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "environment variable exists",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "env_value",
			expected:     "env_value",
		},
		{
			name:         "environment variable does not exist",
			key:          "NONEXISTENT_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.envValue != "" {
				os.Setenv(test.key, test.envValue)
				defer os.Unsetenv(test.key)
			} else {
				os.Unsetenv(test.key)
			}

			result := getEnvOrDefault(test.key, test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected '%s', got '%s'", test.expected, result)
			}
		})
	}
}

func TestGetEnvOrDefaultInt(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{
			name:         "valid integer environment variable",
			key:          "TEST_INT_KEY",
			defaultValue: 100,
			envValue:     "50",
			expected:     50,
		},
		{
			name:         "invalid integer environment variable",
			key:          "TEST_INT_KEY",
			defaultValue: 100,
			envValue:     "invalid",
			expected:     100,
		},
		{
			name:         "missing environment variable",
			key:          "NONEXISTENT_INT_KEY",
			defaultValue: 100,
			envValue:     "",
			expected:     100,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.envValue != "" {
				os.Setenv(test.key, test.envValue)
				defer os.Unsetenv(test.key)
			} else {
				os.Unsetenv(test.key)
			}

			result := getEnvOrDefaultInt(test.key, test.defaultValue)
			if result != test.expected {
				t.Errorf("Expected %d, got %d", test.expected, result)
			}
		})
	}
}
