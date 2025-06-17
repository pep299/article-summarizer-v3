package rss

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// RedditComment represents a Reddit comment
type RedditComment struct {
	Body       string                `json:"body"`
	Author     string                `json:"author"`
	Score      int                   `json:"score"`
	CreatedUTC float64               `json:"created_utc"`
	Replies    *RedditCommentListing `json:"replies,omitempty"`
}

// RedditCommentData wraps comment data
type RedditCommentData struct {
	Kind string      `json:"kind"`
	Data interface{} `json:"data"`
}

// RedditCommentListing represents a listing of comments
type RedditCommentListing struct {
	Kind string `json:"kind"`
	Data struct {
		Children []RedditCommentData `json:"children"`
	} `json:"data"`
}

// RedditAPIResponse represents the structure of Reddit JSON API response
type RedditAPIResponse []RedditCommentListing

type RedditRSSRepository struct {
	rssRepo repository.RSSRepository
}

func NewRedditRSSRepository(rssRepo repository.RSSRepository) *RedditRSSRepository {
	return &RedditRSSRepository{
		rssRepo: rssRepo,
	}
}

func (r *RedditRSSRepository) FetchArticles(ctx context.Context) ([]repository.Item, error) {
	url := "https://www.reddit.com/r/programming/.rss"
	headers := map[string]string{
		"User-Agent": "Article Summarizer Bot/1.0 (Reddit)",
		"Accept":     "application/rss+xml, application/xml, text/xml",
	}

	xmlContent, err := r.rssRepo.FetchFeedXML(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("fetching Reddit RSS: %w", err)
	}

	return r.parseFeed(xmlContent)
}

func (r *RedditRSSRepository) FetchComments(ctx context.Context, commentURL string) (*Comments, error) {
	// Convert Reddit post URL to JSON API URL with sort=top parameter
	jsonURL := strings.Replace(commentURL, "reddit.com", "reddit.com", 1) + ".json?sort=top&limit=100"

	// Set headers for JSON API access
	headers := map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Accept":     "application/json",
	}

	// Fetch JSON data
	jsonContent, err := r.rssRepo.FetchFeedXML(ctx, jsonURL, headers)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Reddit comments: %w", err)
	}

	// Parse JSON response
	var apiResponse RedditAPIResponse
	if err := json.Unmarshal([]byte(jsonContent), &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit JSON response: %w", err)
	}

	// Extract comments (skip the first element which is the post itself)
	if len(apiResponse) < 2 {
		return nil, fmt.Errorf("unexpected Reddit API response structure")
	}

	comments := r.extractCommentsFromListing(&apiResponse[1])
	combinedText := r.combineCommentsText(comments)

	return &Comments{Text: combinedText}, nil
}

func (r *RedditRSSRepository) parseFeed(xmlContent string) ([]repository.Item, error) {
	// Reddit uses Atom format
	var atomFeed struct {
		Entries []struct {
			Title string `xml:"title"`
			Link  struct {
				Href string `xml:"href,attr"`
			} `xml:"link"`
			Content string `xml:"content"`
			Updated string `xml:"updated"`
			ID      string `xml:"id"`
		} `xml:"entry"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &atomFeed); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit Atom format: %w", err)
	}

	var items []repository.Item
	for _, entry := range atomFeed.Entries {
		redditURL := entry.Link.Href

		// Extract external URL from Reddit post content
		externalURL := r.extractExternalURL(entry.Content)

		// Extract external URL, fallback to reddit URL
		linkURL := externalURL
		if linkURL == "" {
			linkURL = redditURL
		}

		parsedDate, _ := r.parseDate(entry.Updated)

		item := repository.Item{
			Title:       entry.Title,
			Link:        linkURL,
			Description: entry.Content,
			PubDate:     entry.Updated,
			GUID:        entry.ID,
			ParsedDate:  parsedDate,
			Source:      "reddit",
			CommentURL:  redditURL,
		}

		items = append(items, item)
	}

	return items, nil
}

func (r *RedditRSSRepository) extractExternalURL(content string) string {
	// Regex to find URLs in the Reddit post content
	// Look for [link] tags that contain the actual article URL
	linkRegex := regexp.MustCompile(`<span><a href="([^"]+)"[^>]*>\[link\]</a></span>`)
	matches := linkRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		url := matches[1]
		// Skip Reddit URLs (comments, user pages, etc.)
		if !strings.Contains(url, "reddit.com") {
			return url
		}
	}

	// Alternative: look for any external HTTP/HTTPS URL in content
	urlRegex := regexp.MustCompile(`href="(https?://[^"]+)"[^>]*>\[link\]`)
	matches = urlRegex.FindStringSubmatch(content)
	if len(matches) > 1 {
		url := matches[1]
		if !strings.Contains(url, "reddit.com") {
			return url
		}
	}

	return ""
}

func (r *RedditRSSRepository) parseDate(dateStr string) (time.Time, error) {
	// Reddit Atom uses RFC3339 format
	formats := []string{
		time.RFC3339, // Atom standard
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05-07:00",
		time.RFC1123Z, // Fallback
		time.RFC1123,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse Reddit date: %s", dateStr)
}

// extractCommentsFromListing recursively extracts comments from Reddit API response
func (r *RedditRSSRepository) extractCommentsFromListing(listing *RedditCommentListing) []RedditComment {
	var comments []RedditComment

	for _, child := range listing.Data.Children {
		if child.Kind == "t1" { // t1 indicates a comment
			// Parse the comment data
			commentData, err := json.Marshal(child.Data)
			if err != nil {
				continue
			}

			var comment RedditComment
			if err := json.Unmarshal(commentData, &comment); err != nil {
				continue
			}

			// Skip deleted/removed comments
			if comment.Body == "[deleted]" || comment.Body == "[removed]" || comment.Body == "" {
				continue
			}

			comments = append(comments, comment)

			// Recursively extract replies
			if comment.Replies != nil {
				replies := r.extractCommentsFromListing(comment.Replies)
				comments = append(comments, replies...)
			}
		}
	}

	return comments
}

// combineCommentsText combines all comment texts into a single string for summarization
func (r *RedditRSSRepository) combineCommentsText(comments []RedditComment) string {
	var parts []string
	totalLength := 0
	const maxLength = 10000 // 10KB limit

	// Reddit API with sort=top provides score-sorted comments
	for _, comment := range comments {
		commentText := fmt.Sprintf("[Score: %d] %s: %s",
			comment.Score, comment.Author, comment.Body)

		if totalLength+len(commentText)+2 > maxLength {
			break
		}

		parts = append(parts, commentText)
		totalLength += len(commentText) + 2
	}

	return strings.Join(parts, "\n\n")
}
