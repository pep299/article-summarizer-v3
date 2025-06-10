package repository

import (
	"context"
	"fmt"
	"io"
	"net/http"
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
	FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error)
	GetUniqueItems(items []Item) []Item
}

type rssRepository struct {
	httpClient *http.Client
}

func NewRSSRepository() RSSRepository {
	return &rssRepository{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (r *rssRepository) FetchFeedXML(ctx context.Context, url string, headers map[string]string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Set headers from strategy
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response body: %w", err)
	}

	return string(body), nil
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
