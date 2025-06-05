package rss

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestParseRSSDate(t *testing.T) {
	tests := []struct {
		input    string
		expected bool // whether parsing should succeed
	}{
		{"Mon, 02 Jan 2006 15:04:05 MST", true},
		{"Mon, 2 Jan 2006 15:04:05 -0700", true},
		{"2006-01-02T15:04:05Z07:00", true},
		{"invalid date", false},
		{"", false},
	}

	for _, test := range tests {
		_, err := parseRSSDate(test.input)
		if test.expected && err != nil {
			t.Errorf("Expected parsing to succeed for '%s', but got error: %v", test.input, err)
		}
		if !test.expected && err == nil {
			t.Errorf("Expected parsing to fail for '%s', but it succeeded", test.input)
		}
	}
}

func TestGetUniqueItems(t *testing.T) {
	items := []Item{
		{Title: "Article 1", Link: "http://example.com/1", GUID: "guid1"},
		{Title: "Article 2", Link: "http://example.com/2", GUID: "guid2"},
		{Title: "Article 1 Duplicate", Link: "http://example.com/1", GUID: "guid1"},
		{Title: "Article 3", Link: "http://example.com/3", GUID: ""},
		{Title: "Article 3 Duplicate", Link: "http://example.com/3", GUID: ""},
	}

	unique := GetUniqueItems(items)

	if len(unique) != 3 {
		t.Errorf("Expected 3 unique items, got %d", len(unique))
	}

	// Check that duplicates were removed correctly
	titles := make(map[string]bool)
	for _, item := range unique {
		if titles[item.Title] {
			t.Errorf("Found duplicate title: %s", item.Title)
		}
		titles[item.Title] = true
	}
}

func TestFilterItems(t *testing.T) {
	now := time.Now()
	items := []Item{
		{
			Title:      "Ask Question",
			Category:   []string{"ask"},
			ParsedDate: now.Add(-1 * time.Hour),
		},
		{
			Title:      "Valid Article",
			Category:   []string{"tech"},
			ParsedDate: now.Add(-1 * time.Hour),
		},
		{
			Title:      "Old Article",
			Category:   []string{"tech"},
			ParsedDate: now.Add(-25 * time.Hour),
		},
		{
			Title: "Short", // Too short
		},
	}

	options := FilterOptions{
		ExcludeCategories: []string{"ask"},
		MaxAge:            24 * time.Hour,
		MinTitleLength:    10,
	}

	filtered := FilterItems(items, options)

	if len(filtered) != 1 {
		t.Errorf("Expected 1 filtered item, got %d", len(filtered))
	}

	if filtered[0].Title != "Valid Article" {
		t.Errorf("Expected 'Valid Article', got '%s'", filtered[0].Title)
	}
}

func TestShouldIncludeItem(t *testing.T) {
	now := time.Now()
	
	tests := []struct {
		name     string
		item     Item
		options  FilterOptions
		expected bool
	}{
		{
			name: "valid item",
			item: Item{
				Title:      "Valid Article Title",
				Category:   []string{"tech"},
				ParsedDate: now.Add(-1 * time.Hour),
			},
			options: FilterOptions{
				ExcludeCategories: []string{"ask"},
				MaxAge:            24 * time.Hour,
				MinTitleLength:    10,
			},
			expected: true,
		},
		{
			name: "excluded category",
			item: Item{
				Title:      "Ask Question Title",
				Category:   []string{"ask"},
				ParsedDate: now.Add(-1 * time.Hour),
			},
			options: FilterOptions{
				ExcludeCategories: []string{"ask"},
				MaxAge:            24 * time.Hour,
				MinTitleLength:    10,
			},
			expected: false,
		},
		{
			name: "too old",
			item: Item{
				Title:      "Old Article Title",
				Category:   []string{"tech"},
				ParsedDate: now.Add(-25 * time.Hour),
			},
			options: FilterOptions{
				ExcludeCategories: []string{"ask"},
				MaxAge:            24 * time.Hour,
				MinTitleLength:    10,
			},
			expected: false,
		},
		{
			name: "title too short",
			item: Item{
				Title:      "Short",
				Category:   []string{"tech"},
				ParsedDate: now.Add(-1 * time.Hour),
			},
			options: FilterOptions{
				ExcludeCategories: []string{"ask"},
				MaxAge:            24 * time.Hour,
				MinTitleLength:    10,
			},
			expected: false,
		},
		{
			name: "excluded keyword in title",
			item: Item{
				Title:       "Article about spam content",
				Description: "Valid description",
			},
			options: FilterOptions{
				ExcludeKeywords: []string{"spam"},
			},
			expected: false,
		},
		{
			name: "excluded keyword in description",
			item: Item{
				Title:       "Valid title",
				Description: "This is spam content",
			},
			options: FilterOptions{
				ExcludeKeywords: []string{"spam"},
			},
			expected: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := shouldIncludeItem(test.item, test.options)
			if result != test.expected {
				t.Errorf("Expected %v, got %v", test.expected, result)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	
	if client == nil {
		t.Error("Expected non-nil client")
	}
	
	if client.httpClient == nil {
		t.Error("Expected non-nil http client")
	}
	
	if client.userAgent == "" {
		t.Error("Expected non-empty user agent")
	}
	
	if !strings.Contains(client.userAgent, "article-summarizer-v3") {
		t.Errorf("Expected user agent to contain 'article-summarizer-v3', got '%s'", client.userAgent)
	}
}

func TestFetchMultipleFeedsWithLimit(t *testing.T) {
	client := NewClient()
	ctx := context.Background()
	
	// Test with invalid URLs to check error handling
	urls := []string{
		"invalid-url",
		"http://nonexistent.example.com/rss",
	}
	
	feeds, errors := client.FetchMultipleFeedsWithLimit(ctx, urls, 2)
	
	if len(feeds) != 0 {
		t.Errorf("Expected 0 successful feeds, got %d", len(feeds))
	}
	
	if len(errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(errors))
	}
	
	// Test with empty URLs
	feeds2, errors2 := client.FetchMultipleFeedsWithLimit(ctx, []string{}, 2)
	
	if len(feeds2) != 0 {
		t.Errorf("Expected 0 feeds for empty URLs, got %d", len(feeds2))
	}
	
	if len(errors2) != 0 {
		t.Errorf("Expected 0 errors for empty URLs, got %d", len(errors2))
	}
}
