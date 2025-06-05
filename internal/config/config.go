package config

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
	SlackBotToken   string `json:"-"` // Don't expose in JSON
	SlackChannel    string `json:"slack_channel"`
	WebhookSlackChannel string `json:"webhook_slack_channel"`

	// RSS settings
	RSSFeeds map[string]RSSFeedConfig `json:"rss_feeds"`

	// Cache settings
	CacheType     string `json:"cache_type"`     // "memory"
	CacheDuration int    `json:"cache_duration"` // in hours
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
		Port:         getEnvOrDefault("PORT", "8080"),
		Host:         getEnvOrDefault("HOST", "0.0.0.0"),
		GeminiAPIKey: getEnvOrDefault("GEMINI_API_KEY", ""),
		GeminiModel:  getEnvOrDefault("GEMINI_MODEL", "gemini-2.5-flash"),
		SlackBotToken: getEnvOrDefault("SLACK_BOT_TOKEN", ""),
		SlackChannel:    getEnvOrDefault("SLACK_CHANNEL", "#article-summarizer"),
		WebhookSlackChannel: getEnvOrDefault("WEBHOOK_SLACK_CHANNEL", "#ondemand-article-summary"),
		CacheType:       getEnvOrDefault("CACHE_TYPE", "memory"),
		CacheDuration:   getEnvOrDefaultInt("CACHE_DURATION_HOURS", 24),
		RSSFeeds: map[string]RSSFeedConfig{
			"hatena": {
				Name:     "はてブ テクノロジー",
				URL:      "https://b.hatena.ne.jp/hotentry/it.rss",
				Enabled:  true,
				Schedule: "0 */30 * * * *", // every 30 minutes
			},
			"lobsters": {
				Name:     "Lobsters",
				URL:      "https://lobste.rs/rss",
				Enabled:  true,
				Schedule: "0 */45 * * * *", // every 45 minutes
			},
		},
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
	if c.CacheDuration <= 0 {
		return &ConfigError{Field: "CACHE_DURATION_HOURS", Message: "must be positive"}
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
