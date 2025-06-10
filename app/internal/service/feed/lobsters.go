package feed

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// LobstersStrategy implements strategy for Lobsters RSS feed
type LobstersStrategy struct{}

func NewLobstersStrategy() *LobstersStrategy {
	return &LobstersStrategy{}
}

func (l *LobstersStrategy) GetConfig() FeedConfig {
	return FeedConfig{
		Name:        "lobsters",
		URL:         "https://lobste.rs/rss",
		DisplayName: "Lobsters",
	}
}

func (l *LobstersStrategy) FilterItems(items []repository.Item) []repository.Item {
	var filtered []repository.Item

	for _, item := range items {
		// Filter out "ask" category for Lobsters
		shouldInclude := true
		for _, category := range item.Category {
			if strings.EqualFold(category, "ask") {
				shouldInclude = false
				break
			}
		}

		if shouldInclude {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

func (l *LobstersStrategy) ParseFeed(xmlContent string) ([]repository.Item, error) {
	// Lobsters uses standard RSS 2.0 format
	var rss struct {
		Channel struct {
			Items []repository.Item `xml:"item"`
		} `xml:"channel"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &rss); err != nil {
		return nil, fmt.Errorf("failed to parse Lobsters RSS 2.0 format: %w", err)
	}

	return rss.Channel.Items, nil
}

func (l *LobstersStrategy) GetRequestHeaders() map[string]string {
	return map[string]string{
		"User-Agent": "Article Summarizer Bot/1.0 (Lobsters)",
		"Accept":     "application/rss+xml, application/xml, text/xml",
	}
}

func (l *LobstersStrategy) ParseDate(dateStr string) (time.Time, error) {
	// Lobsters uses standard RSS date formats
	formats := []string{
		time.RFC1123Z,     // Most common for RSS
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		time.RFC3339,      // Sometimes used
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse Lobsters date: %s", dateStr)
}

