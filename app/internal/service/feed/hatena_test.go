package feed

import (
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

func TestHatenaStrategy(t *testing.T) {
	strategy := NewHatenaStrategy()

	config := strategy.GetConfig()
	if config.Name != "hatena" {
		t.Errorf("Expected name 'hatena', got '%s'", config.Name)
	}
	if config.URL != "https://b.hatena.ne.jp/hotentry/it.rss" {
		t.Errorf("Expected Hatena URL, got '%s'", config.URL)
	}
	if config.DisplayName != "はてブ テクノロジー" {
		t.Errorf("Expected DisplayName 'はてブ テクノロジー', got '%s'", config.DisplayName)
	}

	// Test filtering - Hatena should not filter anything
	items := []repository.Item{
		{Title: "Article 1", Category: []string{"tech"}},
		{Title: "Article 2", Category: []string{"ask"}},
		{Title: "Article 3", Category: []string{"general"}},
	}

	filtered := strategy.FilterItems(items)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 articles after filtering, got %d", len(filtered))
	}
}
