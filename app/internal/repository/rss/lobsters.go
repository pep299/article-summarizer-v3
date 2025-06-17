package rss

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

type LobstersRSSRepository struct {
	rssRepo repository.RSSRepository
}

func NewLobstersRSSRepository(rssRepo repository.RSSRepository) *LobstersRSSRepository {
	return &LobstersRSSRepository{
		rssRepo: rssRepo,
	}
}

func (l *LobstersRSSRepository) FetchArticles(ctx context.Context) ([]repository.Item, error) {
	url := "https://lobste.rs/rss"
	headers := map[string]string{
		"User-Agent": "Article Summarizer Bot/1.0 (Lobsters)",
		"Accept":     "application/rss+xml, application/xml, text/xml",
	}

	xmlContent, err := l.rssRepo.FetchFeedXML(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("fetching Lobsters RSS: %w", err)
	}

	items, err := l.parseFeed(xmlContent)
	if err != nil {
		return nil, err
	}

	return l.filterItems(items), nil
}

func (l *LobstersRSSRepository) FetchComments(ctx context.Context, articleURL string) (*Comments, error) {
	// Convert article URL to Lobsters JSON API URL
	// Example: https://lobste.rs/s/abc123/title -> https://lobste.rs/s/abc123/title.json
	jsonURL := articleURL + ".json"

	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Accept":     "application/json",
	}

	// Fetch JSON data
	jsonContent, err := l.rssRepo.FetchFeedXML(ctx, jsonURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Lobsters JSON: %w", err)
	}

	// Parse JSON response
	var lobstersData LobstersStoryResponse
	if err := json.Unmarshal([]byte(jsonContent), &lobstersData); err != nil {
		return nil, fmt.Errorf("failed to parse Lobsters JSON: %w", err)
	}

	// Convert to Comments format
	var commentTexts []string
	extractCommentsRecursively(lobstersData.Comments, &commentTexts)

	if len(commentTexts) == 0 {
		return &Comments{Text: ""}, nil
	}

	// Combine all comments into a single text for summarization
	combinedText := fmt.Sprintf("以下はLobstersのコメントです:\n\n%s",
		strings.Join(commentTexts, "\n\n"))

	return &Comments{Text: combinedText}, nil
}

func (l *LobstersRSSRepository) parseFeed(xmlContent string) ([]repository.Item, error) {
	// Lobsters uses standard RSS 2.0 format
	var rss struct {
		Channel struct {
			Items []struct {
				Title       string   `xml:"title"`
				Link        string   `xml:"link"`
				Description string   `xml:"description"`
				PubDate     string   `xml:"pubDate"`
				GUID        string   `xml:"guid"`
				Category    []string `xml:"category"`
			} `xml:"item"`
		} `xml:"channel"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &rss); err != nil {
		return nil, fmt.Errorf("failed to parse Lobsters RSS 2.0 format: %w", err)
	}

	var items []repository.Item
	for _, item := range rss.Channel.Items {
		parsedDate, _ := l.parseDate(item.PubDate)

		items = append(items, repository.Item{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     item.PubDate,
			GUID:        item.GUID,
			Category:    item.Category,
			ParsedDate:  parsedDate,
			Source:      "lobsters",
			CommentURL:  "", // Lobsters doesn't have separate comment URLs
		})
	}

	return items, nil
}

func (l *LobstersRSSRepository) filterItems(items []repository.Item) []repository.Item {
	var filtered []repository.Item

	for _, item := range items {
		// Filter out "ask" category for Lobsters
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

func (l *LobstersRSSRepository) parseDate(dateStr string) (time.Time, error) {
	// Lobsters uses standard RSS date formats
	formats := []string{
		time.RFC1123Z, // Most common for RSS
		time.RFC1123,
		"Mon, 2 Jan 2006 15:04:05 -0700",
		"Mon, 2 Jan 2006 15:04:05 MST",
		time.RFC3339, // Sometimes used
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse Lobsters date: %s", dateStr)
}

// LobstersStoryResponse represents the JSON response from Lobsters API
type LobstersStoryResponse struct {
	Title    string            `json:"title"`
	URL      string            `json:"url"`
	Comments []LobstersComment `json:"comments"`
}

type LobstersComment struct {
	Comment     string            `json:"comment"`
	CommentHTML string            `json:"comment_html"`
	CreatedAt   string            `json:"created_at"`
	User        string            `json:"user"`
	Score       int               `json:"score"`
	Replies     []LobstersComment `json:"replies"`
}

// extractCommentsRecursively extracts all comment text from nested structure
func extractCommentsRecursively(comments []LobstersComment, commentTexts *[]string) {
	for _, comment := range comments {
		if strings.TrimSpace(comment.Comment) != "" {
			*commentTexts = append(*commentTexts, comment.Comment)
		}
		// Recursively extract replies
		if len(comment.Replies) > 0 {
			extractCommentsRecursively(comment.Replies, commentTexts)
		}
	}
}
