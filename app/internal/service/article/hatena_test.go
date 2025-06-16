package article

import (
	"context"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/mocks"
	"github.com/pep299/article-summarizer-v3/internal/repository"
)

func TestHatenaProcessor_New(t *testing.T) {
	processor := NewHatenaProcessor(
		&mocks.MockRSSRepo{},
		&mocks.MockGeminiRepo{},
		&mocks.MockSlackRepo{},
		&mocks.MockProcessedRepo{},
		&mocks.MockLimiter{},
	)

	if processor == nil {
		t.Error("Expected non-nil processor")
	}
}

func TestFilterUnprocessedArticles(t *testing.T) {
	ctx := context.Background()
	processedRepo := &mocks.MockProcessedRepo{}

	articles := []repository.Item{
		{Title: "Test Article 1", Link: "https://example.com/1"},
		{Title: "Test Article 2", Link: "https://example.com/2"},
	}

	unprocessed, err := filterUnprocessedArticles(ctx, processedRepo, articles)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(unprocessed) != 2 {
		t.Errorf("Expected 2 unprocessed articles, got %d", len(unprocessed))
	}
}
