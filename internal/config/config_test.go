package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Set some test environment variables
	os.Setenv("GEMINI_API_KEY", "test-key")
	os.Setenv("SLACK_WEBHOOK_URL", "https://hooks.slack.com/test")
	defer os.Unsetenv("GEMINI_API_KEY")
	defer os.Unsetenv("SLACK_WEBHOOK_URL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.GeminiAPIKey != "test-key" {
		t.Errorf("Expected GeminiAPIKey to be 'test-key', got '%s'", cfg.GeminiAPIKey)
	}

	if cfg.SlackWebhookURL != "https://hooks.slack.com/test" {
		t.Errorf("Expected SlackWebhookURL to be 'https://hooks.slack.com/test', got '%s'", cfg.SlackWebhookURL)
	}

	if cfg.Port != "8080" {
		t.Errorf("Expected Port to be '8080', got '%s'", cfg.Port)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test missing required fields
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("SLACK_WEBHOOK_URL")

	_, err := Load()
	if err == nil {
		t.Error("Expected validation error for missing required fields")
	}
}

func TestParseStringSlice(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b , c ", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
	}

	for _, test := range tests {
		result := parseStringSlice(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("For input '%s', expected length %d, got %d", test.input, len(test.expected), len(result))
			continue
		}
		for i, expected := range test.expected {
			if result[i] != expected {
				t.Errorf("For input '%s', expected[%d] = '%s', got '%s'", test.input, i, expected, result[i])
			}
		}
	}
}
