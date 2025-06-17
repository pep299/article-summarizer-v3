package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Hatena RSS Repository
type MockHatenaRSSRepo struct{}

func (m *MockHatenaRSSRepo) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
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
