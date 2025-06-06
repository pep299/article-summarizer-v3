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

// HatenaFeed represents a Hatena RSS feed
type HatenaFeed struct {
	Title       string       `xml:"channel>title"`
	Description string       `xml:"channel>description"`
	Link        string       `xml:"channel>link"`
	Items       []HatenaItem `xml:"channel>item"`
}

// LobstersFeed represents a Lobsters RSS feed
type LobstersFeed struct {
	Title       string         `xml:"channel>title"`
	Description string         `xml:"channel>description"`
	Link        string         `xml:"channel>link"`
	Items       []LobstersItem `xml:"channel>item"`
}

// RSSItem represents a common interface for all RSS items
type RSSItem interface {
	GetTitle() string
	GetLink() string
	GetDescription() string
	GetPubDate() string
	GetParsedDate() time.Time
	GetUniqueID() string // GUID, Link, or other unique identifier
	GetSource() string
	GetCategories() []string
	SetSource(source string)
	SetParsedDate(date time.Time)
}

// Item represents a generic RSS item (backward compatibility)
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

// Implement RSSItem interface for generic Item
func (i *Item) GetTitle() string             { return i.Title }
func (i *Item) GetLink() string              { return i.Link }
func (i *Item) GetDescription() string       { return i.Description }
func (i *Item) GetPubDate() string           { return i.PubDate }
func (i *Item) GetParsedDate() time.Time     { return i.ParsedDate }
func (i *Item) GetSource() string            { return i.Source }
func (i *Item) GetCategories() []string      { return i.Category }
func (i *Item) SetSource(source string)      { i.Source = source }
func (i *Item) SetParsedDate(date time.Time) { i.ParsedDate = date }

func (i *Item) GetUniqueID() string {
	if i.GUID != "" {
		return i.GUID
	}
	return i.Link
}

// HatenaItem represents a Hatena Bookmark RSS item
type HatenaItem struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`     // Always present in Hatena
	Category    []string  `xml:"category"` // Categories available
	ParsedDate  time.Time `xml:"-"`
	Source      string    `xml:"-"`
}

// Implement RSSItem interface for HatenaItem
func (h *HatenaItem) GetTitle() string             { return h.Title }
func (h *HatenaItem) GetLink() string              { return h.Link }
func (h *HatenaItem) GetDescription() string       { return h.Description }
func (h *HatenaItem) GetPubDate() string           { return h.PubDate }
func (h *HatenaItem) GetParsedDate() time.Time     { return h.ParsedDate }
func (h *HatenaItem) GetSource() string            { return h.Source }
func (h *HatenaItem) GetCategories() []string      { return h.Category }
func (h *HatenaItem) SetSource(source string)      { h.Source = source }
func (h *HatenaItem) SetParsedDate(date time.Time) { h.ParsedDate = date }

func (h *HatenaItem) GetUniqueID() string {
	// Hatena always has GUID, but fallback to Link just in case
	if h.GUID != "" {
		return h.GUID
	}
	return h.Link
}

// LobstersItem represents a Lobsters RSS item
type LobstersItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	// No GUID field - Lobsters doesn't provide it
	// No Category field - Lobsters uses tags differently
	ParsedDate time.Time `xml:"-"`
	Source     string    `xml:"-"`
}

// Implement RSSItem interface for LobstersItem
func (l *LobstersItem) GetTitle() string             { return l.Title }
func (l *LobstersItem) GetLink() string              { return l.Link }
func (l *LobstersItem) GetDescription() string       { return l.Description }
func (l *LobstersItem) GetPubDate() string           { return l.PubDate }
func (l *LobstersItem) GetParsedDate() time.Time     { return l.ParsedDate }
func (l *LobstersItem) GetSource() string            { return l.Source }
func (l *LobstersItem) GetCategories() []string      { return []string{} } // No categories in Lobsters
func (l *LobstersItem) SetSource(source string)      { l.Source = source }
func (l *LobstersItem) SetParsedDate(date time.Time) { l.ParsedDate = date }

func (l *LobstersItem) GetUniqueID() string {
	// Lobsters uses Link as unique identifier
	return l.Link
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
