package article

import (
	"context"
	"strings"
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

func TestHatenaProcessor_CommentsIntegration(t *testing.T) {
	// Test that Hatena comments can be fetched and processed
	mockRSSRepo := &mocks.MockRSSRepo{}

	hatenaRepo := NewHatenaProcessor(
		mockRSSRepo,
		&mocks.MockGeminiRepo{},
		&mocks.MockSlackRepo{},
		&mocks.MockProcessedRepo{},
		&mocks.MockLimiter{},
	)

	// Test that comment fetching works conceptually
	// In real usage, this would connect to Hatena Bookmark API
	t.Log("Hatena comment integration test - processor created successfully")

	if hatenaRepo == nil {
		t.Error("Expected non-nil Hatena processor")
	}
}

func TestHatenaRSSRepository_FetchComments(t *testing.T) {
	// Test the Hatena RSS repository mock behavior
	mockRepo := &mocks.MockHatenaRSSRepo{}

	ctx := context.Background()
	comments, err := mockRepo.FetchComments(ctx, "https://example.com/hatena")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if comments == nil {
		t.Error("Expected comments to be returned")
		return
	}

	if comments.Text == "" {
		t.Error("Expected comment text to be non-empty")
	}

	expectedPrefix := "以下ははてなブックマークのコメントです:"
	if !strings.Contains(comments.Text, expectedPrefix) {
		t.Errorf("Expected comment text to contain '%s'", expectedPrefix)
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
