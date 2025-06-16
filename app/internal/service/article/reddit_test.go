package article

import (
	"context"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/mocks"
)

func TestRedditProcessor_New(t *testing.T) {
	processor := NewRedditProcessor(
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

func TestRedditProcessor_Process_NoArticles(t *testing.T) {
	processor := NewRedditProcessor(
		&mocks.MockRSSRepo{},
		&mocks.MockGeminiRepo{},
		&mocks.MockSlackRepo{},
		&mocks.MockProcessedRepo{},
		&mocks.MockLimiter{},
	)

	ctx := context.Background()
	err := processor.Process(ctx)

	if err != nil {
		t.Errorf("Expected no error with no articles, got %v", err)
	}
}
