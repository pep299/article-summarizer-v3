package mocks

import (
	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Limiter
type MockLimiter struct{}

func (m *MockLimiter) Limit(articles []repository.Item) []repository.Item {
	return articles
}
