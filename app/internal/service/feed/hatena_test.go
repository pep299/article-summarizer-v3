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

// TestHatenaStrategy_DateParsing tests Hatena-specific date parsing
// This tests hatena.go:51-67 ParseDate() for RFC3339 format handling
func TestHatenaStrategy_DateParsing(t *testing.T) {
	strategy := NewHatenaStrategy()
	
	tests := []struct {
		name        string
		inputDate   string
		expectError bool
		description string
	}{
		{
			name:        "Valid RFC3339",
			inputDate:   "2024-01-15T10:30:00+09:00",
			expectError: false,
			description: "Standard Hatena date format should parse",
		},
		{
			name:        "Invalid date",
			inputDate:   "invalid-date",
			expectError: true,
			description: "Invalid date should return error",
		},
		{
			name:        "Empty date",
			inputDate:   "",
			expectError: true,
			description: "Empty date should return error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedTime, err := strategy.ParseDate(tt.inputDate)

			if tt.expectError {
				if err == nil {
					t.Errorf("%s: expected error but got none", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("%s: unexpected error: %v", tt.description, err)
				}
				if parsedTime.IsZero() {
					t.Errorf("%s: expected valid time, got zero time", tt.description)
				}
			}
		})
	}
}
