package mocks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/repository/rss"
)

// Mock Hatena RSS Repository
type MockHatenaRSSRepo struct {
	ShouldFailComments bool
}

func (m *MockHatenaRSSRepo) FetchArticles(ctx context.Context) ([]repository.Item, error) {
	return []repository.Item{
		{
			Title:       "Test Hatena Article",
			Link:        "https://example.com/hatena",
			Description: "Test description",
			PubDate:     "Mon, 01 Jan 2024 00:00:00 GMT",
			GUID:        "https://example.com/hatena",
			Source:      "hatena",
		},
	}, nil
}

func (m *MockHatenaRSSRepo) FetchComments(ctx context.Context, articleURL string) (*rss.Comments, error) {
	if m.ShouldFailComments {
		return nil, fmt.Errorf("mock comment fetch error")
	}

	return &rss.Comments{
		Text: "以下ははてなブックマークのコメントです:\n\nGreat article!\n\nVery informative",
	}, nil
}

func (m *MockHatenaRSSRepo) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
	// Mock Hatena Bookmark API response
	if url == "https://b.hatena.ne.jp/entry/jsonlite/?url=https%3A//example.com/hatena" {
		response := map[string]interface{}{
			"bookmarks": []map[string]interface{}{
				{
					"comment":   "Great article!",
					"user":      "user1",
					"timestamp": "2024-01-01T00:00:00Z",
					"tags":      []string{"tech", "programming"},
				},
				{
					"comment":   "Very informative",
					"user":      "user2",
					"timestamp": "2024-01-01T01:00:00Z",
					"tags":      []string{"tech"},
				},
			},
			"count": 2,
			"url":   "https://example.com/hatena",
			"title": "Test Hatena Article",
		}
		jsonBytes, _ := json.Marshal(response)
		return string(jsonBytes), nil
	}

	// Default RSS feed response
	return `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0">
		<channel>
			<item>
				<title>Test Hatena Article</title>
				<link>https://example.com/hatena</link>
				<description>Test description</description>
				<pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
				<guid>https://example.com/hatena</guid>
			</item>
		</channel>
	</rss>`, nil
}

func (m *MockHatenaRSSRepo) GetUniqueItems(items []repository.Item) []repository.Item {
	return items
}
