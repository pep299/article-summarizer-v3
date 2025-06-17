package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock RSS Repository
type MockRSSRepo struct {
	Articles []repository.Item
}

func (m *MockRSSRepo) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
	// Return empty RSS feed
	return `<?xml version="1.0" encoding="UTF-8"?>
	<rss version="2.0">
		<channel>
		</channel>
	</rss>`, nil
}

func (m *MockRSSRepo) GetUniqueItems(items []repository.Item) []repository.Item {
	return items
}
