package mocks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/repository/rss"
)

// Mock Lobsters RSS Repository
type MockLobstersRSSRepo struct {
	ShouldFailComments bool
}

func (m *MockLobstersRSSRepo) FetchArticles(ctx context.Context) ([]repository.Item, error) {
	return []repository.Item{
		{
			Title:       "Test Lobsters Article",
			Link:        "https://lobste.rs/s/abc123/test",
			Description: "Test description",
			PubDate:     "Mon, 01 Jan 2024 00:00:00 GMT",
			GUID:        "https://lobste.rs/s/abc123/test",
			Source:      "lobsters",
			Category:    []string{"programming"},
		},
	}, nil
}

func (m *MockLobstersRSSRepo) FetchComments(ctx context.Context, articleURL string) (*rss.Comments, error) {
	if m.ShouldFailComments {
		return nil, fmt.Errorf("mock comment fetch error")
	}

	return &rss.Comments{
		Text: "以下はLobstersのコメントです:\n\nInteresting discussion\n\nThanks for sharing\n\nYou're welcome!",
	}, nil
}

func (m *MockLobstersRSSRepo) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
	// Mock Lobsters JSON API response
	if url == "https://lobste.rs/s/abc123/test.json" {
		response := map[string]interface{}{
			"title": "Test Lobsters Article",
			"url":   "https://example.com/article",
			"comments": []map[string]interface{}{
				{
					"comment":    "Interesting discussion",
					"user":       "user1",
					"created_at": "2024-01-01T00:00:00Z",
					"score":      5,
					"replies":    []interface{}{},
				},
				{
					"comment":    "Thanks for sharing",
					"user":       "user2",
					"created_at": "2024-01-01T01:00:00Z",
					"score":      3,
					"replies": []map[string]interface{}{
						{
							"comment":    "You're welcome!",
							"user":       "user1",
							"created_at": "2024-01-01T02:00:00Z",
							"score":      2,
							"replies":    []interface{}{},
						},
					},
				},
			},
		}
		jsonBytes, _ := json.Marshal(response)
		return string(jsonBytes), nil
	}

	// Default RSS feed response
	return `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0">
		<channel>
			<item>
				<title>Test Lobsters Article</title>
				<link>https://lobste.rs/s/abc123/test</link>
				<description>Test description</description>
				<pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
				<guid>https://lobste.rs/s/abc123/test</guid>
				<category>programming</category>
			</item>
		</channel>
	</rss>`, nil
}

func (m *MockLobstersRSSRepo) GetUniqueItems(items []repository.Item) []repository.Item {
	return items
}
