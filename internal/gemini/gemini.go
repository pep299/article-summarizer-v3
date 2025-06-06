package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/rss"
)

// Client handles Gemini API operations
type Client struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Gemini API client
func NewClient(apiKey, model string) *Client {
	return &Client{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// SummarizeRequest represents a summarization request
type SummarizeRequest struct {
	Title       string
	Link        string
	Description string
	Content     string
}

// SummarizeResponse represents a summarization response
type SummarizeResponse struct {
	Summary     string    `json:"summary"`
	ProcessedAt time.Time `json:"processed_at"`
}

// geminiRequest represents the request structure for Gemini API
type geminiRequest struct {
	Contents         []geminiContent         `json:"contents"`
	GenerationConfig *geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature"`
	TopP            float64 `json:"topP"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}

// geminiResponse represents the response structure from Gemini API
type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

// SummarizeURL summarizes a URL by fetching content and sending to Gemini (RSS mode)
func (c *Client) SummarizeURL(ctx context.Context, url string) (*SummarizeResponse, error) {
	// Fetch HTML content
	htmlContent, err := c.fetchHTML(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching HTML: %w", err)
	}

	// Extract text from HTML
	textContent := c.extractTextFromHTML(htmlContent)
	if textContent == "" {
		return nil, fmt.Errorf("no text content found")
	}

	// Create prompt for RSS mode (shorter summary for team sharing)
	prompt := c.buildRSSPrompt(textContent)

	// Call Gemini API
	summary, err := c.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &SummarizeResponse{
		Summary:     summary,
		ProcessedAt: time.Now(),
	}, nil
}

// SummarizeURLForOnDemand summarizes a URL for on-demand requests (longer summary)
func (c *Client) SummarizeURLForOnDemand(ctx context.Context, url string) (*SummarizeResponse, error) {
	// Fetch HTML content
	htmlContent, err := c.fetchHTML(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetching HTML: %w", err)
	}

	// Extract text from HTML
	textContent := c.extractTextFromHTML(htmlContent)
	if textContent == "" {
		return nil, fmt.Errorf("no text content found")
	}

	// Create prompt for on-demand mode (longer summary)
	prompt := c.buildOnDemandPrompt(textContent)

	// Call Gemini API
	summary, err := c.callGeminiAPI(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &SummarizeResponse{
		Summary:     summary,
		ProcessedAt: time.Now(),
	}, nil
}

// fetchHTML fetches HTML content from a URL
func (c *Client) fetchHTML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Article Summarizer Bot/1.0)")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching URL: %w", err)
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

// extractTextFromHTML extracts text content from HTML (same as v1)
func (c *Client) extractTextFromHTML(html string) string {
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

// buildRSSPrompt creates a prompt for RSS mode (same as v1)
func (c *Client) buildRSSPrompt(textContent string) string {
	// Limit content to 10KB
	if len(textContent) > 10000 {
		textContent = textContent[:10000]
	}

	return fmt.Sprintf(`以下のテキストを、Slackチャンネルでチームメンバーが素早く理解できるよう、1000文字以内で簡潔に要約してください。

**重要な制約:**
- 推測や創作は一切せず、実際に記載されている内容のみを要約してください
- 記載されていない情報は追加しないでください

チーム共有を前提とした読みやすい形式で、以下の構造で出力してください：
- 📝 **要約:** 実際の内容を3-4行で簡潔に
- 🎯 **対象者:** 記載されている課題や対象を基に
- 💡 **解決効果:** 明記されている効果や解決策のみ

テキスト内容:
%s`, textContent)
}

// buildOnDemandPrompt creates a prompt for on-demand mode (same as v1)
func (c *Client) buildOnDemandPrompt(textContent string) string {
	// Limit content to 10KB
	if len(textContent) > 10000 {
		textContent = textContent[:10000]
	}

	return fmt.Sprintf(`以下のテキストを、Slackチャンネルでチームメンバーが素早く理解できるよう、800-1200文字程度（短すぎず長すぎない適切な分量）で要約してください。

**重要な制約:**
- 推測や創作は一切せず、実際に記載されている内容のみを要約してください
- 記載されていない情報は追加しないでください

チーム共有を前提とした読みやすい形式で、以下の構造で出力してください：
- 📝 **要約:** 実際の内容を3-4行で簡潔に
- 🎯 **対象者:** 記載されている課題や対象を基に
- 💡 **解決効果:** 明記されている効果や解決策のみ

テキスト内容:
%s`, textContent)
}

// callGeminiAPI makes the actual API call to Gemini
func (c *Client) callGeminiAPI(ctx context.Context, prompt string) (string, error) {
	geminiReq := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &geminiGenerationConfig{
			Temperature:     0.3,
			TopP:            0.8,
			MaxOutputTokens: 8000,
		},
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// ConvertRSSItemToRequest converts an RSS item to a summarization request
func ConvertRSSItemToRequest(item rss.Item) SummarizeRequest {
	return SummarizeRequest{
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     "", // Content will be fetched separately
	}
}
