package infrastructure

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	Port string `json:"port"`
	Host string `json:"host"`

	// Gemini API settings
	GeminiAPIKey string `json:"-"` // Don't expose in JSON
	GeminiModel  string `json:"gemini_model"`

	// Slack settings
	SlackBotToken       string `json:"-"` // Don't expose in JSON
	SlackChannel        string `json:"slack_channel"`
	WebhookSlackChannel string `json:"webhook_slack_channel"`

	// Webhook settings
	WebhookAuthToken string `json:"-"` // Don't expose in JSON

	// RSS settings
	HatenaRSSURL   string `json:"hatena_rss_url"`
	LobstersRSSURL string `json:"lobsters_rss_url"`
}

// RSSFeedConfig represents configuration for a single RSS feed
type RSSFeedConfig struct {
	Name     string `json:"name"`
	URL      string `json:"url"`
	Enabled  bool   `json:"enabled"`
	Schedule string `json:"schedule"` // cron expression for individual scheduling
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	config := &Config{
		Port:                "8080",
		Host:                "0.0.0.0",
		GeminiAPIKey:        getEnvOrDefault("GEMINI_API_KEY", ""),
		GeminiModel:         "gemini-2.5-flash-preview-05-20",
		SlackBotToken:       getEnvOrDefault("SLACK_BOT_TOKEN", ""),
		SlackChannel:        "#dev-null",
		WebhookSlackChannel: "#dev-null",
		WebhookAuthToken:    getEnvOrDefault("WEBHOOK_AUTH_TOKEN", ""),
		HatenaRSSURL:        "https://b.hatena.ne.jp/hotentry/it.rss",
		LobstersRSSURL:      "https://lobste.rs/rss",
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

// getEnvOrDefaultInt returns environment variable value as int or default if not set
func getEnvOrDefaultInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
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
