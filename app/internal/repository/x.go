package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// PostData represents data from a social media post
type PostData struct {
	AuthorName string    `json:"author_name"`
	AuthorURL  string    `json:"author_url"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
	URL        string    `json:"url"`
}

// Client defines the interface for fetching social media posts
type Client interface {
	// FetchPost extracts post data from the given URL
	FetchPost(ctx context.Context, url string) (*PostData, error)

	// FetchQuoteChain fetches a chain of quoted posts starting from the given URL
	FetchQuoteChain(ctx context.Context, url string) ([]PostData, error)

	// IsSupported checks if the client supports the given URL
	IsSupported(url string) bool
}

// XClient implements the Client interface for X (Twitter) posts
type XClient struct {
	httpClient *http.Client
	oembedURL  string
}

// oEmbedResponse represents the response from X's oEmbed API
type oEmbedResponse struct {
	URL          string `json:"url"`
	AuthorName   string `json:"author_name"`
	AuthorURL    string `json:"author_url"`
	HTML         string `json:"html"`
	Width        int    `json:"width"`
	Height       *int   `json:"height"`
	Type         string `json:"type"`
	CacheAge     string `json:"cache_age"`
	ProviderName string `json:"provider_name"`
	ProviderURL  string `json:"provider_url"`
	Version      string `json:"version"`
}

// NewXClient creates a new X (Twitter) client
func NewXClient() *XClient {
	return &XClient{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		oembedURL: "https://publish.twitter.com/oembed",
	}
}

// IsSupported checks if the URL is a supported X (Twitter) URL
func (x *XClient) IsSupported(url string) bool {
	// Support both x.com and twitter.com domains
	xURLPattern := regexp.MustCompile(`^https?://(www\.)?(x\.com|twitter\.com)/.+/status/\d+`)
	return xURLPattern.MatchString(url)
}

// FetchPost extracts post data from the given X (Twitter) URL
func (x *XClient) FetchPost(ctx context.Context, postURL string) (*PostData, error) {
	if !x.IsSupported(postURL) {
		return nil, fmt.Errorf("unsupported URL format: %s", postURL)
	}

	// Call oEmbed API
	oembedResp, err := x.callOEmbedAPI(ctx, postURL)
	if err != nil {
		return nil, fmt.Errorf("oEmbed API call failed: %w", err)
	}

	// Extract text from HTML
	text, err := x.extractTextFromHTML(oembedResp.HTML)
	if err != nil {
		return nil, fmt.Errorf("failed to extract text from HTML: %w", err)
	}

	// Extract creation date from HTML
	createdAt, err := x.extractCreatedAtFromHTML(oembedResp.HTML)
	if err != nil {
		// Fallback to current time if extraction fails
		createdAt = time.Now()
	}

	return &PostData{
		AuthorName: oembedResp.AuthorName,
		AuthorURL:  oembedResp.AuthorURL,
		Text:       text,
		CreatedAt:  createdAt,
		URL:        oembedResp.URL,
	}, nil
}

// callOEmbedAPI makes a request to X's oEmbed API
func (x *XClient) callOEmbedAPI(ctx context.Context, postURL string) (*oEmbedResponse, error) {
	// Build oEmbed API URL
	apiURL := fmt.Sprintf("%s?url=%s", x.oembedURL, url.QueryEscape(postURL))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Article Summarizer Bot/1.0)")

	resp, err := x.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oEmbed API returned status %d", resp.StatusCode)
	}

	var oembedResp oEmbedResponse
	if err := json.NewDecoder(resp.Body).Decode(&oembedResp); err != nil {
		return nil, fmt.Errorf("failed to decode oEmbed response: %w", err)
	}

	return &oembedResp, nil
}

// FetchQuoteChain fetches a chain of quoted posts starting from the given URL
func (x *XClient) FetchQuoteChain(ctx context.Context, startURL string) ([]PostData, error) {
	if !x.IsSupported(startURL) {
		return nil, fmt.Errorf("unsupported URL format: %s", startURL)
	}

	var chain []PostData
	currentURL := startURL
	visited := make(map[string]bool)

	// Maximum depth to prevent infinite loops
	maxDepth := 10

	for i := 0; i < maxDepth && currentURL != "" && !visited[currentURL]; i++ {
		visited[currentURL] = true

		// Get oEmbed data for current URL
		oembedResp, err := x.callOEmbedAPI(ctx, currentURL)
		if err != nil {
			// If we can't get this tweet, break the chain
			break
		}

		// Extract text and create PostData
		text, err := x.extractTextFromHTML(oembedResp.HTML)
		if err != nil {
			text = "" // Continue with empty text if extraction fails
		}

		createdAt, err := x.extractCreatedAtFromHTML(oembedResp.HTML)
		if err != nil {
			createdAt = time.Now() // Fallback to current time
		}

		postData := PostData{
			AuthorName: oembedResp.AuthorName,
			AuthorURL:  oembedResp.AuthorURL,
			Text:       text,
			CreatedAt:  createdAt,
			URL:        oembedResp.URL,
		}

		// Add to beginning of chain (so final result is chronological order)
		chain = append([]PostData{postData}, chain...)

		// Extract t.co URL from HTML
		tcoURL := x.extractTcoURL(oembedResp.HTML)
		if tcoURL == "" {
			break
		}

		// Expand t.co URL to get actual tweet URL
		expandedURL, err := x.expandTcoURL(ctx, tcoURL)
		if err != nil {
			// If expansion fails, break the chain
			break
		}

		currentURL = expandedURL
	}

	return chain, nil
}

// extractTcoURL extracts t.co URL from HTML
func (x *XClient) extractTcoURL(html string) string {
	tcoPattern := regexp.MustCompile(`https://t\.co/[a-zA-Z0-9]+`)
	return tcoPattern.FindString(html)
}

// expandTcoURL expands a t.co URL to get the actual URL
func (x *XClient) expandTcoURL(ctx context.Context, tcoURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequestWithContext(ctx, "GET", tcoURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to expand t.co URL: %w", err)
	}
	defer resp.Body.Close()

	location := resp.Header.Get("Location")
	if location == "" {
		return "", fmt.Errorf("no redirect found for t.co URL")
	}

	return location, nil
}

// extractTextFromHTML extracts the tweet text from the oEmbed HTML
func (x *XClient) extractTextFromHTML(html string) (string, error) {
	// Extract text from the first <p> tag within the blockquote
	pTagPattern := regexp.MustCompile(`<p[^>]*>(.*?)</p>`)
	matches := pTagPattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find tweet text in HTML")
	}

	text := matches[1]

	// Remove HTML tags and entities
	text = x.cleanHTML(text)

	return strings.TrimSpace(text), nil
}

// extractCreatedAtFromHTML extracts the creation date from the oEmbed HTML
func (x *XClient) extractCreatedAtFromHTML(html string) (time.Time, error) {
	// Look for the date in the format "Month Day, Year"
	datePattern := regexp.MustCompile(`>([A-Za-z]+ \d{1,2}, \d{4})<`)
	matches := datePattern.FindStringSubmatch(html)
	if len(matches) < 2 {
		return time.Time{}, fmt.Errorf("could not find date in HTML")
	}

	dateStr := matches[1]

	// Parse the date string
	parsedTime, err := time.Parse("January 2, 2006", dateStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to parse date %s: %w", dateStr, err)
	}

	return parsedTime, nil
}

// cleanHTML removes HTML tags and decodes HTML entities
func (x *XClient) cleanHTML(text string) string {
	// Remove HTML tags
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	text = tagPattern.ReplaceAllString(text, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	return text
}
