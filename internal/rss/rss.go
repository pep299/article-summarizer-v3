package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
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
		userAgent: "article-summarizer-v3/1.0",
	}
}

// FetchFeed fetches and parses an RSS feed from the given URL
func (c *Client) FetchFeed(ctx context.Context, url string) (*Feed, error) {
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

	var feed Feed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, fmt.Errorf("parsing RSS feed: %w", err)
	}

	// Parse dates for items
	for i := range feed.Items {
		if feed.Items[i].PubDate != "" {
			if parsedDate, err := parseRSSDate(feed.Items[i].PubDate); err == nil {
				feed.Items[i].ParsedDate = parsedDate
			}
		}
	}

	return &feed, nil
}

// FetchMultipleFeeds fetches multiple RSS feeds concurrently
func (c *Client) FetchMultipleFeeds(ctx context.Context, urls []string) (map[string]*Feed, map[string]error) {
	type result struct {
		url  string
		feed *Feed
		err  error
	}

	results := make(chan result, len(urls))
	
	// Start goroutines for each feed
	for _, url := range urls {
		go func(u string) {
			feed, err := c.FetchFeed(ctx, u)
			results <- result{url: u, feed: feed, err: err}
		}(url)
	}

	// Collect results
	feeds := make(map[string]*Feed)
	errors := make(map[string]error)

	for i := 0; i < len(urls); i++ {
		res := <-results
		if res.err != nil {
			errors[res.url] = res.err
		} else {
			feeds[res.url] = res.feed
		}
	}

	return feeds, errors
}

// FilterItems filters RSS items based on criteria
func FilterItems(items []Item, options FilterOptions) []Item {
	var filtered []Item

	for _, item := range items {
		if shouldIncludeItem(item, options) {
			filtered = append(filtered, item)
		}
	}

	return filtered
}

// FilterOptions holds filtering criteria
type FilterOptions struct {
	ExcludeCategories []string
	MinTitleLength    int
	MaxAge            time.Duration
	ExcludeKeywords   []string
}

// shouldIncludeItem checks if an item should be included based on filter options
func shouldIncludeItem(item Item, options FilterOptions) bool {
	// Check title length
	if options.MinTitleLength > 0 && len(item.Title) < options.MinTitleLength {
		return false
	}

	// Check age
	if options.MaxAge > 0 && !item.ParsedDate.IsZero() {
		if time.Since(item.ParsedDate) > options.MaxAge {
			return false
		}
	}

	// Check excluded categories
	for _, category := range item.Category {
		for _, excluded := range options.ExcludeCategories {
			if strings.EqualFold(category, excluded) {
				return false
			}
		}
	}

	// Check excluded keywords in title and description
	titleLower := strings.ToLower(item.Title)
	descLower := strings.ToLower(item.Description)
	
	for _, keyword := range options.ExcludeKeywords {
		keywordLower := strings.ToLower(keyword)
		if strings.Contains(titleLower, keywordLower) || strings.Contains(descLower, keywordLower) {
			return false
		}
	}

	return true
}

// parseRSSDate parses various RSS date formats
func parseRSSDate(dateStr string) (time.Time, error) {
	// Common RSS date formats
	formats := []string{
		time.RFC1123Z,
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		"2006-01-02T15:04:05Z07:00",
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
