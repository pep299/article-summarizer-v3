package rss

import (
	"context"
	"strings"
	"testing"
)

func TestParseRSSDate(t *testing.T) {
	tests := []struct {
		input    string
		expected bool // whether parsing should succeed
	}{
		{"Mon, 02 Jan 2006 15:04:05 MST", true},
		{"Mon, 2 Jan 2006 15:04:05 -0700", true},
		{"2006-01-02T15:04:05Z", true},
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
	items := []Item{
		{
			Title:    "Ask Question",
			Category: []string{"ask"},
		},
		{
			Title:    "Valid Article",
			Category: []string{"tech"},
		},
		{
			Title:    "Another Valid Article",
			Category: []string{"programming"},
		},
		{
			Title:    "Ask HN: Question",
			Category: []string{"ask", "hn"},
		},
	}

	filtered := FilterItems(items)

	if len(filtered) != 2 {
		t.Errorf("Expected 2 filtered items, got %d", len(filtered))
	}

	// Check that "ask" category items were filtered out
	for _, item := range filtered {
		for _, category := range item.Category {
			if strings.EqualFold(category, "ask") {
				t.Errorf("Found item with 'ask' category that should have been filtered: %s", item.Title)
			}
		}
	}

	// Check that valid articles remain
	validFound := false
	for _, item := range filtered {
		if item.Title == "Valid Article" {
			validFound = true
			break
		}
	}
	if !validFound {
		t.Error("Expected 'Valid Article' to remain after filtering")
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

	if !strings.Contains(client.userAgent, "Article Summarizer Bot") {
		t.Errorf("Expected user agent to contain 'Article Summarizer Bot', got '%s'", client.userAgent)
	}
}

func TestFetchFeedErrorHandling(t *testing.T) {
	client := NewClient()
	ctx := context.Background()

	// Test with invalid URL to check error handling
	_, err := client.FetchFeed(ctx, "test", "invalid-url")
	if err == nil {
		t.Error("Expected error for invalid URL")
	}

	// Test with non-existent URL
	_, err = client.FetchFeed(ctx, "test", "http://nonexistent.example.com/rss")
	if err == nil {
		t.Error("Expected error for non-existent URL")
	}
}

func TestParseRSSXML(t *testing.T) {
	client := NewClient()

	// Test RSS 2.0 format
	rss20XML := `<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test Feed</title>
		<item>
			<title>Test Article</title>
			<link>http://example.com/test</link>
			<description>Test description</description>
			<guid>test-guid</guid>
			<category>tech</category>
		</item>
	</channel>
</rss>`

	items, err := client.parseRSSXML(rss20XML)
	if err != nil {
		t.Fatalf("Failed to parse RSS 2.0 XML: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item, got %d", len(items))
	}

	if items[0].Title != "Test Article" {
		t.Errorf("Expected title 'Test Article', got '%s'", items[0].Title)
	}

	// Test RDF format
	rdfXML := `<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#">
	<item>
		<title>RDF Test Article</title>
		<link>http://example.com/rdf-test</link>
		<description>RDF test description</description>
	</item>
</rdf:RDF>`

	items, err = client.parseRSSXML(rdfXML)
	if err != nil {
		t.Fatalf("Failed to parse RDF XML: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("Expected 1 item from RDF, got %d", len(items))
	}

	if items[0].Title != "RDF Test Article" {
		t.Errorf("Expected title 'RDF Test Article', got '%s'", items[0].Title)
	}

	// Test invalid XML
	_, err = client.parseRSSXML("invalid xml content")
	if err == nil {
		t.Error("Expected error for invalid XML")
	}
}


func TestExtractTextFromHTML(t *testing.T) {
	_ = NewClient()

	html := `<html>
	<head>
		<title>Test Page</title>
		<script>console.log('test');</script>
		<style>body { color: red; }</style>
	</head>
	<body>
		<h1>Main Title</h1>
		<p>This is a paragraph with <strong>bold text</strong>.</p>
		<div>Another div with content.</div>
		<script>alert('should be removed');</script>
	</body>
</html>`

	text := extractTextFromHTML(html)

	// Should contain main content
	if !strings.Contains(text, "Main Title") {
		t.Error("Expected extracted text to contain 'Main Title'")
	}

	if !strings.Contains(text, "This is a paragraph") {
		t.Error("Expected extracted text to contain paragraph content")
	}

	if !strings.Contains(text, "bold text") {
		t.Error("Expected extracted text to contain 'bold text'")
	}

	// Should not contain script or style content
	if strings.Contains(text, "console.log") {
		t.Error("Expected extracted text to not contain script content")
	}

	if strings.Contains(text, "color: red") {
		t.Error("Expected extracted text to not contain style content")
	}

	if strings.Contains(text, "alert") {
		t.Error("Expected extracted text to not contain script content")
	}

	// Should not contain HTML tags
	if strings.Contains(text, "<h1>") || strings.Contains(text, "</h1>") {
		t.Error("Expected extracted text to not contain HTML tags")
	}
}
