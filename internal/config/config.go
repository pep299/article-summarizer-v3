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
	SlackWebhookURL string `json:"-"` // Don't expose in JSON
	SlackChannel    string `json:"slack_channel"`

	// RSS settings
	RSSFeeds      []string `json:"rss_feeds"`
	UpdateInterval int     `json:"update_interval_minutes"`

	// Cache settings
	CacheType     string `json:"cache_type"`     // "memory" or "sqlite"
	CacheDuration int    `json:"cache_duration"` // in hours

	// Rate limiting
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
}

// Load reads configuration from environment variables and .env file
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	config := &Config{
		Port:         getEnvOrDefault("PORT", "8080"),
		Host:         getEnvOrDefault("HOST", "0.0.0.0"),
		GeminiAPIKey: getEnvOrDefault("GEMINI_API_KEY", ""),
		GeminiModel:  getEnvOrDefault("GEMINI_MODEL", "gemini-1.5-flash"),
		SlackWebhookURL: getEnvOrDefault("SLACK_WEBHOOK_URL", ""),
		SlackChannel:    getEnvOrDefault("SLACK_CHANNEL", "#general"),
		RSSFeeds:        parseStringSlice(getEnvOrDefault("RSS_FEEDS", "https://b.hatena.ne.jp/hotentry/it.rss,https://lobste.rs/rss")),
		UpdateInterval:  getEnvOrDefaultInt("UPDATE_INTERVAL_MINUTES", 30),
		CacheType:       getEnvOrDefault("CACHE_TYPE", "memory"),
		CacheDuration:   getEnvOrDefaultInt("CACHE_DURATION_HOURS", 24),
		MaxConcurrentRequests: getEnvOrDefaultInt("MAX_CONCURRENT_REQUESTS", 5),
	}

	return config, config.validate()
}

// validate checks if required configuration values are present
func (c *Config) validate() error {
	if c.GeminiAPIKey == "" {
		return &ConfigError{Field: "GEMINI_API_KEY", Message: "Gemini API key is required"}
	}
	if c.SlackWebhookURL == "" {
		return &ConfigError{Field: "SLACK_WEBHOOK_URL", Message: "Slack webhook URL is required"}
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

// parseStringSlice parses comma-separated string into slice
func parseStringSlice(value string) []string {
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// ConfigError represents a configuration error
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return e.Field + ": " + e.Message
}
