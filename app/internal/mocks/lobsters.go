package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Lobsters RSS Repository
type MockLobstersRSSRepo struct{}

func (m *MockLobstersRSSRepo) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
	return `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0">
		<channel>
			<item>
				<title>Test Lobsters Article</title>
				<link>https://lobste.rs/s/test</link>
				<description>Test description</description>
				<pubDate>Mon, 01 Jan 2024 00:00:00 GMT</pubDate>
				<guid>https://lobste.rs/s/test</guid>
			</item>
		</channel>
	</rss>`, nil
}

func (m *MockLobstersRSSRepo) GetUniqueItems(items []repository.Item) []repository.Item {
	return items
}
