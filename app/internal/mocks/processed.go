package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Processed Repository
type MockProcessedRepo struct{}

func (m *MockProcessedRepo) LoadIndex(ctx context.Context) (map[string]*repository.IndexEntry, error) {
	return make(map[string]*repository.IndexEntry), nil
}

func (m *MockProcessedRepo) IsProcessed(key string, index map[string]*repository.IndexEntry) bool {
	return false
}

func (m *MockProcessedRepo) MarkAsProcessed(ctx context.Context, article repository.Item) error {
	return nil
}

func (m *MockProcessedRepo) GenerateKey(article repository.Item) string {
	return article.Link
}

func (m *MockProcessedRepo) Close() error {
	return nil
}
