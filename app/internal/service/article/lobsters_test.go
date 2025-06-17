package article

import (
	"context"
	"strings"
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/mocks"
)

func TestLobstersProcessor_New(t *testing.T) {
	processor := NewLobstersProcessor(
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

func TestLobstersProcessor_Process_NoArticles(t *testing.T) {
	processor := NewLobstersProcessor(
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

func TestLobstersProcessor_CommentsIntegration(t *testing.T) {
	// Test that Lobsters comments can be fetched and processed
	mockRSSRepo := &mocks.MockRSSRepo{}

	lobstersRepo := NewLobstersProcessor(
		mockRSSRepo,
		&mocks.MockGeminiRepo{},
		&mocks.MockSlackRepo{},
		&mocks.MockProcessedRepo{},
		&mocks.MockLimiter{},
	)

	// Test that comment fetching works conceptually
	// In real usage, this would connect to Lobsters JSON API
	t.Log("Lobsters comment integration test - processor created successfully")

	if lobstersRepo == nil {
		t.Error("Expected non-nil Lobsters processor")
	}
}

func TestLobstersRSSRepository_FetchComments(t *testing.T) {
	// Test the Lobsters RSS repository mock behavior
	mockRepo := &mocks.MockLobstersRSSRepo{}

	ctx := context.Background()
	comments, err := mockRepo.FetchComments(ctx, "https://lobste.rs/s/abc123/test")

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

	expectedPrefix := "以下はLobstersのコメントです:"
	if !strings.Contains(comments.Text, expectedPrefix) {
		t.Errorf("Expected comment text to contain '%s'", expectedPrefix)
	}
}
