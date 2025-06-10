package feed

import (
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

func TestLobstersStrategy(t *testing.T) {
	strategy := NewLobstersStrategy()

	config := strategy.GetConfig()
	if config.Name != "lobsters" {
		t.Errorf("Expected name 'lobsters', got '%s'", config.Name)
	}
	if config.URL != "https://lobste.rs/rss" {
		t.Errorf("Expected Lobsters URL, got '%s'", config.URL)
	}
	if config.DisplayName != "Lobsters" {
		t.Errorf("Expected DisplayName 'Lobsters', got '%s'", config.DisplayName)
	}

	// Test filtering - Lobsters should filter out "ask" category
	items := []repository.Item{
		{Title: "Article 1", Category: []string{"tech"}},
		{Title: "Ask: Question", Category: []string{"ask"}},
		{Title: "Article 3", Category: []string{"general"}},
		{Title: "Ask: Another Question", Category: []string{"ASK"}}, // Test case-insensitive
	}

	filtered := strategy.FilterItems(items)
	if len(filtered) != 2 {
		t.Errorf("Expected 2 articles after filtering, got %d", len(filtered))
	}

	// Verify the right articles remained
	expectedTitles := map[string]bool{
		"Article 1": true,
		"Article 3": true,
	}

	for _, article := range filtered {
		if !expectedTitles[article.Title] {
			t.Errorf("Unexpected article after filtering: %s", article.Title)
		}
	}
}
