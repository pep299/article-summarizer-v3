package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Feed represents an RSS feed
type Feed struct {
	Title       string `xml:"channel>title"`
	Description string `xml:"channel>description"`
	Link        string `xml:"channel>link"`
	Items       []Item `xml:"channel>item"`
}

// Item represents an RSS item
type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`
	Category    []string  `xml:"category"`
	ParsedDate  time.Time `xml:"-"`
	Source      string    `xml:"-"` // Added to track source
}

// Client handles RSS feed operations
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient creates a new RSS client
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Article Summarizer Bot/1.0",
	}
}

// FetchFeed fetches and parses an RSS feed from the given URL
func (c *Client) FetchFeed(ctx context.Context, feedName, url string) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	items, err := c.parseRSSXML(string(body))
	if err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	// Set source for all items
	for i := range items {
		items[i].Source = feedName
		// Parse dates for items
		if items[i].PubDate != "" {
			if parsedDate, err := parseRSSDate(items[i].PubDate); err == nil {
				items[i].ParsedDate = parsedDate
			}
		}
	}

	return items, nil
}

// parseRSSXML parses RSS XML content (supports both RSS 2.0 and RDF)
func (c *Client) parseRSSXML(xmlContent string) ([]Item, error) {
	// Try RSS 2.0 format first
	var rss struct {
		Channel struct {
			Items []Item `xml:"item"`
		} `xml:"channel"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &rss); err == nil && len(rss.Channel.Items) > 0 {
		return rss.Channel.Items, nil
	}

	// Try RDF format (used by Hatena Bookmark)
	var rdf struct {
		Items []Item `xml:"item"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &rdf); err == nil && len(rdf.Items) > 0 {
		return rdf.Items, nil
	}

	return nil, fmt.Errorf("unable to parse RSS format")
}

// FilterItems filters RSS items (only removes "ask" category like v1)
func FilterItems(items []Item) []Item {
	var filtered []Item

	for _, item := range items {
		// Only filter out "ask" category (same as v1 Lobsters filtering)
		shouldInclude := true
		for _, category := range item.Category {
			if strings.EqualFold(category, "ask") {
				shouldInclude = false
				break
			}
		}

		if shouldInclude {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// parseRSSDate parses various RSS date formats
func parseRSSDate(dateStr string) (time.Time, error) {
	// Common RSS date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		time.RFC3339,
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", dateStr)
}

// GetUniqueItems removes duplicate items based on GUID or link
func GetUniqueItems(items []Item) []Item {
	seen := make(map[string]bool)
	var unique []Item

	for _, item := range items {
		key := item.GUID
		if key == "" {
			key = item.Link
		}

		if key != "" && !seen[key] {
			seen[key] = true
			unique = append(unique, item)
		}
	}

	return unique
}

// generateKey generates a cache key for an RSS item
func generateKey(item Item) string {
	// Use GUID if available, otherwise use link
	identifier := item.GUID
	if identifier == "" {
		identifier = item.Link
	}

	// Create a simple hash-like key
	return fmt.Sprintf("article:%s", identifier)
}

// extractTextFromHTML extracts text content from HTML (for testing)
func extractTextFromHTML(html string) string {
	// Remove script and style tags
	scriptRe := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	html = scriptRe.ReplaceAllString(html, "")

	styleRe := regexp.MustCompile(`(?i)<style[^>]*>[\s\S]*?</style>`)
	html = styleRe.ReplaceAllString(html, "")

	// Remove HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text := tagRe.ReplaceAllString(html, " ")

	// Normalize whitespace
	spaceRe := regexp.MustCompile(`\s+`)
	text = spaceRe.ReplaceAllString(text, " ")

	return strings.TrimSpace(text)
}
