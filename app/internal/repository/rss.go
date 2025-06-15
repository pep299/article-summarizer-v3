package repository

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

// Item represents an RSS item
type Item struct {
	Title       string    `xml:"title"`
	Link        string    `xml:"link"`        // 記事の実際のURL（外部URL）
	Description string    `xml:"description"`
	PubDate     string    `xml:"pubDate"`
	GUID        string    `xml:"guid"`
	Category    []string  `xml:"category"`
	ParsedDate  time.Time `xml:"-"`
	Source      string    `xml:"-"`
	CommentURL  string    `xml:"-"`           // コメント/ディスカッションのURL
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
	logger := log.New(funcframework.LogWriter(ctx), "", 0)

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
		logger.Printf("Error making HTTP request to RSS feed %s: %v\nStack:\n%s", url, err, debug.Stack())
		return "", fmt.Errorf("fetching feed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Printf("RSS feed request failed url=%s status_code=%d\nStack:\n%s", url, resp.StatusCode, debug.Stack())
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
