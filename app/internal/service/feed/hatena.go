package feed

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// HatenaStrategy implements strategy for Hatena RSS feed
type HatenaStrategy struct{}

func NewHatenaStrategy() *HatenaStrategy {
	return &HatenaStrategy{}
}

func (h *HatenaStrategy) GetConfig() FeedConfig {
	return FeedConfig{
		Name:        "hatena",
		URL:         "https://b.hatena.ne.jp/hotentry/it.rss",
		DisplayName: "はてブ テクノロジー",
	}
}

func (h *HatenaStrategy) FilterItems(items []repository.Item) []repository.Item {
	// Hatena doesn't need special filtering, return all items
	return items
}

func (h *HatenaStrategy) ParseFeed(xmlContent string) ([]repository.Item, error) {
	// Hatena uses RDF format
	var rdf struct {
		Items []repository.Item `xml:"item"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &rdf); err != nil {
		return nil, fmt.Errorf("failed to parse Hatena RDF format: %w", err)
	}

	return rdf.Items, nil
}

func (h *HatenaStrategy) GetRequestHeaders() map[string]string {
	return map[string]string{
		"User-Agent": "Article Summarizer Bot/1.0 (Hatena)",
		"Accept":     "application/rdf+xml, application/xml, text/xml",
	}
}

func (h *HatenaStrategy) ParseDate(dateStr string) (time.Time, error) {
	// Hatena uses standard RFC formats
	formats := []string{
		time.RFC3339,      // Most common for Hatena
		time.RFC1123Z,     
		time.RFC1123,
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse Hatena date: %s", dateStr)
}
