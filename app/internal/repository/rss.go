package repository

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Item represents an RSS item
type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`
	Category    []string  `xml:"category"`
	ParsedDate  time.Time `xml:"-"`
	Source      string    `xml:"-"`
}

func (i *Item) GetUniqueID() string {
	if i.GUID != "" {
		return i.GUID
	}
	return i.Link
}

type RSSRepository interface {
	FetchFeed(ctx context.Context, feedName, url string) ([]Item, error)
	GetUniqueItems(items []Item) []Item
	FilterItems(items []Item) []Item
}

type rssRepository struct {
	httpClient *http.Client
	userAgent  string
}

func NewRSSRepository() RSSRepository {
	return &rssRepository{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Article Summarizer Bot/1.0",
	}
}

func (r *rssRepository) FetchFeed(ctx context.Context, feedName, url string) ([]Item, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", r.userAgent)
	req.Header.Set("Accept", "application/rss+xml, application/xml, text/xml")

	resp, err := r.httpClient.Do(req)
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

	items, err := r.parseRSSXML(string(body))
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

func (r *rssRepository) parseRSSXML(xmlContent string) ([]Item, error) {
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

func (r *rssRepository) GetUniqueItems(items []Item) []Item {
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

func (r *rssRepository) FilterItems(items []Item) []Item {
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
