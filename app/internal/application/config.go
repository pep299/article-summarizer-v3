package application

import (
	"os"
	"strings"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	Port string `json:"port"`
	Host string `json:"host"`

	// Gemini API settings
	GeminiAPIKey  string `json:"-"` // Don't expose in JSON
	GeminiModel   string `json:"gemini_model"`
	GeminiBaseURL string `json:"gemini_base_url"` // For testing

	// Slack settings
	SlackBotToken        string `json:"-"` // Don't expose in JSON
	SlackChannel         string `json:"slack_channel"`
	SlackChannelReddit   string `json:"slack_channel_reddit"`
	SlackChannelHatena   string `json:"slack_channel_hatena"`
	SlackChannelLobsters string `json:"slack_channel_lobsters"`
	WebhookSlackChannel  string `json:"webhook_slack_channel"`
	SlackBaseURL         string `json:"slack_base_url"` // For testing

	// Webhook settings
	WebhookAuthToken string `json:"-"` // Don't expose in JSON
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		Port:                 "8080",
		Host:                 "0.0.0.0",
		GeminiAPIKey:         getEnvOrDefault("GEMINI_API_KEY", ""),
		GeminiModel:          "gemini-2.5-flash-preview-05-20",
		GeminiBaseURL:        getEnvOrDefault("GEMINI_BASE_URL", "https://generativelanguage.googleapis.com/v1beta/models"),
		SlackBotToken:        getEnvOrDefault("SLACK_BOT_TOKEN", ""),
		SlackChannel:         getEnvOrDefault("SLACK_CHANNEL", "#article-summarizer"),
		SlackChannelReddit:   getEnvOrDefault("SLACK_CHANNEL_REDDIT", "#reddit-article-summary"),
		SlackChannelHatena:   getEnvOrDefault("SLACK_CHANNEL_HATENA", "#hatena-article-summary"),
		SlackChannelLobsters: getEnvOrDefault("SLACK_CHANNEL_LOBSTERS", "#lobsters-article-summary"),
		WebhookSlackChannel:  getEnvOrDefault("WEBHOOK_SLACK_CHANNEL", "#ondemand-article-summary"),
		SlackBaseURL:         getEnvOrDefault("SLACK_BASE_URL", "https://slack.com/api"),
		WebhookAuthToken:     getEnvOrDefault("WEBHOOK_AUTH_TOKEN", ""),
	}

	return config, config.validate()
}

// validate checks if required configuration values are present
func (c *Config) validate() error {
	if c.GeminiAPIKey == "" {
		return &ConfigError{Field: "GEMINI_API_KEY", Message: "Gemini API key is required"}
	}
	if c.SlackBotToken == "" {
		return &ConfigError{Field: "SLACK_BOT_TOKEN", Message: "Slack bot token is required"}
	}
	if !strings.HasPrefix(c.SlackBotToken, "xoxb-") {
		return &ConfigError{Field: "SLACK_BOT_TOKEN", Message: "must start with xoxb-"}
	}
	return nil
}

// getEnvOrDefault returns environment variable value or default if not set
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}
