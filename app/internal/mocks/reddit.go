package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Reddit RSS Repository
type MockRedditRSSRepo struct{}

func (m *MockRedditRSSRepo) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
	return `<?xml version="1.0" encoding="UTF-8"?>
	<feed xmlns="http://www.w3.org/2005/Atom">
		<entry>
			<title>Test Reddit Post</title>
			<link href="https://www.reddit.com/r/programming/comments/test"/>
			<content>Test content with external link</content>
			<updated>2024-01-01T00:00:00Z</updated>
			<id>t3_test</id>
		</entry>
	</feed>`, nil
}

func (m *MockRedditRSSRepo) GetUniqueItems(items []repository.Item) []repository.Item {
	return items
}
