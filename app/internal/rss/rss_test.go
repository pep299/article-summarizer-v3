package rss

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// Test successful RSS feed fetch with mock server
func TestFetchFeed_Success(t *testing.T) {
	// Create mock RSS server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "GET" {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		userAgent := r.Header.Get("User-Agent")
		if !strings.Contains(userAgent, "Article Summarizer Bot") {
			t.Errorf("Expected User-Agent to contain 'Article Summarizer Bot', got '%s'", userAgent)
		}

		accept := r.Header.Get("Accept")
		if !strings.Contains(accept, "application/rss+xml") {
			t.Errorf("Expected Accept header to contain RSS MIME type, got '%s'", accept)
		}

		// Return mock RSS feed
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Test RSS Feed</title>
		<description>Test feed description</description>
		<link>http://example.com</link>
		<item>
			<title>Test Article 1</title>
			<link>http://example.com/article1</link>
			<description>Description of article 1</description>
			<pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate>
			<guid>http://example.com/article1</guid>
			<category>tech</category>
		</item>
		<item>
			<title>Test Article 2</title>
			<link>http://example.com/article2</link>
			<description>Description of article 2</description>
			<pubDate>Tue, 03 Jan 2006 10:30:00 +0000</pubDate>
			<guid>http://example.com/article2</guid>
			<category>news</category>
		</item>
	</channel>
</rss>`))
	}))
	defer mockServer.Close()

	client := NewClient()
	ctx := context.Background()

	items, err := client.FetchFeed(ctx, "Test Feed", mockServer.URL)
	if err != nil {
		t.Fatalf("Expected successful feed fetch, got error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Verify first item
	item1 := items[0]
	if item1.Title != "Test Article 1" {
		t.Errorf("Expected title 'Test Article 1', got '%s'", item1.Title)
	}
	if item1.Link != "http://example.com/article1" {
		t.Errorf("Expected link 'http://example.com/article1', got '%s'", item1.Link)
	}
	if item1.Source != "Test Feed" {
		t.Errorf("Expected source 'Test Feed', got '%s'", item1.Source)
	}
	if item1.GUID != "http://example.com/article1" {
		t.Errorf("Expected GUID 'http://example.com/article1', got '%s'", item1.GUID)
	}

	// Verify parsed date is set
	if item1.ParsedDate.IsZero() {
		t.Error("Expected parsed date to be set")
	}

	// Verify categories
	if len(item1.Category) == 0 || item1.Category[0] != "tech" {
		t.Errorf("Expected category 'tech', got %v", item1.Category)
	}
}

func TestFetchFeed_RDFFormat(t *testing.T) {
	// Create mock RSS server with RDF format
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rdf+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rdf:RDF xmlns:rdf="http://www.w3.org/1999/02/22-rdf-syntax-ns#"
         xmlns="http://purl.org/rss/1.0/">
	<item>
		<title>RDF Article 1</title>
		<link>http://example.com/rdf1</link>
		<description>RDF article description</description>
		<category>technology</category>
	</item>
	<item>
		<title>RDF Article 2</title>
		<link>http://example.com/rdf2</link>
		<description>Another RDF article</description>
		<category>programming</category>
	</item>
</rdf:RDF>`))
	}))
	defer mockServer.Close()

	client := NewClient()
	ctx := context.Background()

	items, err := client.FetchFeed(ctx, "RDF Feed", mockServer.URL)
	if err != nil {
		t.Fatalf("Expected successful RDF feed fetch, got error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("Expected 2 items, got %d", len(items))
	}

	// Verify RDF items
	item1 := items[0]
	if item1.Title != "RDF Article 1" {
		t.Errorf("Expected title 'RDF Article 1', got '%s'", item1.Title)
	}
	if item1.Source != "RDF Feed" {
		t.Errorf("Expected source 'RDF Feed', got '%s'", item1.Source)
	}
}

func TestFetchFeed_HTTPError(t *testing.T) {
	// Create mock server that returns HTTP error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer mockServer.Close()

	client := NewClient()
	ctx := context.Background()

	_, err := client.FetchFeed(ctx, "Test Feed", mockServer.URL)
	if err == nil {
		t.Error("Expected error for HTTP 404, but got none")
	}

	if !strings.Contains(err.Error(), "404") {
		t.Errorf("Expected error to mention status code, got: %v", err)
	}
}

func TestFetchFeed_InvalidXML(t *testing.T) {
	// Create mock server that returns invalid XML
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<invalid>xml content without proper RSS structure</invalid>`))
	}))
	defer mockServer.Close()

	client := NewClient()
	ctx := context.Background()

	_, err := client.FetchFeed(ctx, "Test Feed", mockServer.URL)
	if err == nil {
		t.Error("Expected error for invalid RSS XML, but got none")
	}

	if !strings.Contains(err.Error(), "unable to parse RSS format") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestFetchFeed_EmptyResponse(t *testing.T) {
	// Create mock server that returns empty response
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(""))
	}))
	defer mockServer.Close()

	client := NewClient()
	ctx := context.Background()

	_, err := client.FetchFeed(ctx, "Empty Feed", mockServer.URL)
	if err == nil {
		t.Error("Expected error for empty response, but got none")
	}
}

func TestFetchFeed_LargeResponse(t *testing.T) {
	// Create mock server that returns large RSS feed
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/rss+xml")
		w.WriteHeader(http.StatusOK)

		// Build large RSS feed
		w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
	<channel>
		<title>Large Test Feed</title>
		<description>Feed with many items</description>
		<link>http://example.com</link>`))

		// Add many items
		for i := 0; i < 100; i++ {
			item := fmt.Sprintf(`
		<item>
			<title>Test Article %d</title>
			<link>http://example.com/article%d</link>
			<description>Description of article %d</description>
			<pubDate>Mon, 02 Jan 2006 15:04:05 +0000</pubDate>
			<guid>http://example.com/article%d</guid>
			<category>tech</category>
		</item>`, i, i, i, i)
			w.Write([]byte(item))
		}

		w.Write([]byte(`
	</channel>
</rss>`))
	}))
	defer mockServer.Close()

	client := NewClient()
	ctx := context.Background()

	items, err := client.FetchFeed(ctx, "Large Feed", mockServer.URL)
	if err != nil {
		t.Fatalf("Expected successful feed fetch for large feed, got error: %v", err)
	}

	if len(items) != 100 {
		t.Errorf("Expected 100 items, got %d", len(items))
	}
}

func TestExtractTextFromHTML_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
		contains []string
		excludes []string
	}{
		{
			name:     "Empty HTML",
			html:     "",
			expected: "",
		},
		{
			name:     "Plain text",
			html:     "Just plain text",
			contains: []string{"Just plain text"},
		},
		{
			name:     "HTML entities",
			html:     "<p>Text with &amp; &lt; &gt; &quot; entities</p>",
			contains: []string{"Text with", "entities"},
		},
		{
			name:     "Nested tags",
			html:     "<div><p><span><strong>Deeply nested</strong> text</span></p></div>",
			contains: []string{"Deeply nested text"},
			excludes: []string{"<div>", "<p>", "<span>", "<strong>"},
		},
		{
			name:     "Mixed content",
			html:     "<h1>Title</h1><script>alert('bad');</script><p>Good content</p><style>.bad{}</style>",
			contains: []string{"Title", "Good content"},
			excludes: []string{"alert", "bad", ".bad{}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTextFromHTML(tt.html)

			if tt.expected != "" && result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}

			for _, contain := range tt.contains {
				if !strings.Contains(result, contain) {
					t.Errorf("Expected result to contain %q, got %q", contain, result)
				}
			}

			for _, exclude := range tt.excludes {
				if strings.Contains(result, exclude) {
					t.Errorf("Expected result to not contain %q, got %q", exclude, result)
				}
			}
		})
	}
}

func TestParseRSSDate_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"RFC1123 format", "Mon, 02 Jan 2006 15:04:05 GMT", true},
		{"RFC1123Z format", "Mon, 02 Jan 2006 15:04:05 +0000", true},
		{"ISO8601 format", "2006-01-02T15:04:05Z", true},
		{"ISO8601 with timezone", "2006-01-02T15:04:05+07:00", true},
		{"Single digit day", "Mon, 2 Jan 2006 15:04:05 GMT", true},
		{"Malformed timezone", "Mon, 02 Jan 2006 15:04:05 BadTZ", false},
		{"Incomplete date", "Mon, 02 Jan", false},
		{"Wrong format", "January 2, 2006", false},
		{"Numbers only", "20060102150405", false},
		{"Whitespace only", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseRSSDate(tt.input)
			if tt.expected && err != nil {
				t.Errorf("Expected parsing to succeed for '%s', but got error: %v", tt.input, err)
			}
			if !tt.expected && err == nil {
				t.Errorf("Expected parsing to fail for '%s', but it succeeded", tt.input)
			}
		})
	}
}

func TestGetUniqueItems_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		items    []Item
		expected int
	}{
		{
			name:     "Empty slice",
			items:    []Item{},
			expected: 0,
		},
		{
			name: "All unique items",
			items: []Item{
				{Title: "A", Link: "http://a.com", GUID: "a"},
				{Title: "B", Link: "http://b.com", GUID: "b"},
				{Title: "C", Link: "http://c.com", GUID: "c"},
			},
			expected: 3,
		},
		{
			name: "All duplicate items by GUID",
			items: []Item{
				{Title: "Same", Link: "http://a.com", GUID: "same"},
				{Title: "Same", Link: "http://b.com", GUID: "same"},
				{Title: "Same", Link: "http://c.com", GUID: "same"},
			},
			expected: 1,
		},
		{
			name: "Mix of GUID and Link duplicates",
			items: []Item{
				{Title: "A", Link: "http://same.com", GUID: "a"},
				{Title: "B", Link: "http://same.com", GUID: ""},
				{Title: "C", Link: "http://different.com", GUID: "a"},
			},
			expected: 2, // Link duplicate removed, GUID duplicate kept (first occurrence)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUniqueItems(tt.items)
			if len(result) != tt.expected {
				t.Errorf("Expected %d unique items, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestFilterItems_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		items    []Item
		expected int
	}{
		{
			name:     "Empty slice",
			items:    []Item{},
			expected: 0,
		},
		{
			name: "No ask categories",
			items: []Item{
				{Title: "Tech Article", Category: []string{"tech", "programming"}},
				{Title: "News Article", Category: []string{"news"}},
			},
			expected: 2,
		},
		{
			name: "All ask categories",
			items: []Item{
				{Title: "Ask HN", Category: []string{"ask"}},
				{Title: "Ask Question", Category: []string{"ASK", "question"}},
			},
			expected: 0,
		},
		{
			name: "Mixed case ask",
			items: []Item{
				{Title: "Valid", Category: []string{"tech"}},
				{Title: "Ask Item", Category: []string{"Ask"}},
				{Title: "ASK Item", Category: []string{"ASK"}},
				{Title: "ask item", Category: []string{"ask"}},
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterItems(tt.items)
			if len(result) != tt.expected {
				t.Errorf("Expected %d filtered items, got %d", tt.expected, len(result))
			}
		})
	}
}
