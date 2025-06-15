package feed

import (
	"testing"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

func TestRedditStrategy(t *testing.T) {
	strategy := NewRedditStrategy()

	config := strategy.GetConfig()
	if config.Name != "reddit" {
		t.Errorf("Expected name 'reddit', got '%s'", config.Name)
	}
	if config.URL != "https://www.reddit.com/r/programming/.rss" {
		t.Errorf("Expected Reddit URL, got '%s'", config.URL)
	}
	if config.DisplayName != "Reddit r/programming" {
		t.Errorf("Expected DisplayName 'Reddit r/programming', got '%s'", config.DisplayName)
	}

	// Test filtering - Reddit now includes all items
	items := []repository.Item{
		{Title: "Article 1", Link: "https://example.com/article1", CommentURL: "https://reddit.com/r/programming/1"},
		{Title: "Self Post", Link: "https://reddit.com/r/programming/2", CommentURL: "https://reddit.com/r/programming/2"},
		{Title: "Article 3", Link: "https://example.com/article3", CommentURL: "https://reddit.com/r/programming/3"},
	}

	filtered := strategy.FilterItems(items)
	if len(filtered) != 3 {
		t.Errorf("Expected 3 articles after filtering, got %d", len(filtered))
	}
}

// TestRedditStrategy_DateParsing tests Reddit-specific date parsing
func TestRedditStrategy_DateParsing(t *testing.T) {
	strategy := NewRedditStrategy()

	tests := []struct {
		name        string
		inputDate   string
		expectError bool
		description string
	}{
		{
			name:        "Valid RFC1123Z",
			inputDate:   "Mon, 02 Jan 2006 15:04:05 -0700",
			expectError: false,
			description: "Standard RSS date format should parse",
		},
		{
			name:        "Valid RFC1123",
			inputDate:   "Mon, 02 Jan 2006 15:04:05 MST",
			expectError: false,
			description: "RFC1123 format should parse",
		},
		{
			name:        "Valid RFC3339",
			inputDate:   "2006-01-02T15:04:05Z",
			expectError: false,
			description: "RFC3339 format should parse",
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

// TestRedditStrategy_ExtractExternalURL tests URL extraction from Reddit content
func TestRedditStrategy_ExtractExternalURL(t *testing.T) {
	strategy := NewRedditStrategy()

	tests := []struct {
		name        string
		content     string
		expectedURL string
		description string
	}{
		{
			name:        "External link with [link] tag",
			content:     `submitted by <a href="https://www.reddit.com/user/test"> /u/test </a> <br/> <span><a href="https://example.com/article">[link]</a></span>`,
			expectedURL: "https://example.com/article",
			description: "Should extract external URL from [link] tag",
		},
		{
			name:        "Reddit link should be ignored",
			content:     `submitted by <a href="https://www.reddit.com/user/test"> /u/test </a> <br/> <span><a href="https://www.reddit.com/r/programming/comments/abc">[link]</a></span>`,
			expectedURL: "",
			description: "Should ignore Reddit URLs",
		},
		{
			name:        "No external link",
			content:     `This is a self post with no external links`,
			expectedURL: "",
			description: "Should return empty string for self posts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.ExtractExternalURL(tt.content)
			if result != tt.expectedURL {
				t.Errorf("%s: expected '%s', got '%s'", tt.description, tt.expectedURL, result)
			}
		})
	}
}
