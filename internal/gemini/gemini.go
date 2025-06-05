package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	KeyPoints   []string  `json:"key_points"`
	Category    string    `json:"category"`
	Confidence  float64   `json:"confidence"`
	ProcessedAt time.Time `json:"processed_at"`
}

// geminiRequest represents the request structure for Gemini API
type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

// geminiResponse represents the response structure from Gemini API
type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

// SummarizeArticle summarizes an article using Gemini API
func (c *Client) SummarizeArticle(ctx context.Context, req SummarizeRequest) (*SummarizeResponse, error) {
	prompt := c.buildPrompt(req)
	
	geminiReq := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", c.baseURL, c.model, c.apiKey)
	
	body, err := json.Marshal(geminiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content in response")
	}

	responseText := geminiResp.Candidates[0].Content.Parts[0].Text
	
	return c.parseResponse(responseText), nil
}

// SummarizeMultipleArticles summarizes multiple articles concurrently
func (c *Client) SummarizeMultipleArticles(ctx context.Context, requests []SummarizeRequest, maxConcurrent int) ([]SummarizeResponse, []error) {
	type result struct {
		index    int
		response *SummarizeResponse
		err      error
	}

	// Create semaphore for rate limiting
	semaphore := make(chan struct{}, maxConcurrent)
	results := make(chan result, len(requests))

	// Start goroutines for each request
	for i, req := range requests {
		go func(index int, request SummarizeRequest) {
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			resp, err := c.SummarizeArticle(ctx, request)
			results <- result{index: index, response: resp, err: err}
		}(i, req)
	}

	// Collect results
	responses := make([]SummarizeResponse, len(requests))
	errors := make([]error, len(requests))

	for i := 0; i < len(requests); i++ {
		res := <-results
		if res.err != nil {
			errors[res.index] = res.err
		} else {
			responses[res.index] = *res.response
		}
	}

	return responses, errors
}

// buildPrompt creates a prompt for the Gemini API
func (c *Client) buildPrompt(req SummarizeRequest) string {
	var content strings.Builder
	
	content.WriteString("記事を要約してください。以下の形式でJSON形式で回答してください：\n\n")
	content.WriteString("{\n")
	content.WriteString("  \"summary\": \"記事の要約（3-4文程度）\",\n")
	content.WriteString("  \"key_points\": [\"重要なポイント1\", \"重要なポイント2\", \"重要なポイント3\"],\n")
	content.WriteString("  \"category\": \"カテゴリ（技術/ビジネス/ニュース等）\",\n")
	content.WriteString("  \"confidence\": 0.85\n")
	content.WriteString("}\n\n")
	
	content.WriteString("記事情報：\n")
	content.WriteString(fmt.Sprintf("タイトル: %s\n", req.Title))
	content.WriteString(fmt.Sprintf("URL: %s\n", req.Link))
	
	if req.Description != "" {
		content.WriteString(fmt.Sprintf("説明: %s\n", req.Description))
	}
	
	if req.Content != "" {
		content.WriteString(fmt.Sprintf("内容: %s\n", req.Content))
	}

	return content.String()
}

// parseResponse parses the Gemini API response
func (c *Client) parseResponse(responseText string) *SummarizeResponse {
	// Try to extract JSON from the response
	start := strings.Index(responseText, "{")
	end := strings.LastIndex(responseText, "}") + 1
	
	if start == -1 || end <= start {
		// Fallback: create a simple summary
		return &SummarizeResponse{
			Summary:     responseText,
			KeyPoints:   []string{},
			Category:    "未分類",
			Confidence:  0.5,
			ProcessedAt: time.Now(),
		}
	}

	jsonStr := responseText[start:end]
	
	var response struct {
		Summary    string   `json:"summary"`
		KeyPoints  []string `json:"key_points"`
		Category   string   `json:"category"`
		Confidence float64  `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &response); err != nil {
		// Fallback: create a simple summary
		return &SummarizeResponse{
			Summary:     responseText,
			KeyPoints:   []string{},
			Category:    "未分類",
			Confidence:  0.5,
			ProcessedAt: time.Now(),
		}
	}

	return &SummarizeResponse{
		Summary:     response.Summary,
		KeyPoints:   response.KeyPoints,
		Category:    response.Category,
		Confidence:  response.Confidence,
		ProcessedAt: time.Now(),
	}
}

// ConvertRSSItemToRequest converts an RSS item to a summarization request
func ConvertRSSItemToRequest(item rss.Item) SummarizeRequest {
	return SummarizeRequest{
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     "", // Content will be fetched separately if needed
	}
}

// SummarizeRSSItems is a convenience function to summarize RSS items
func (c *Client) SummarizeRSSItems(ctx context.Context, items []rss.Item, maxConcurrent int) ([]SummarizeResponse, []error) {
	requests := make([]SummarizeRequest, len(items))
	for i, item := range items {
		requests[i] = ConvertRSSItemToRequest(item)
	}
	
	return c.SummarizeMultipleArticles(ctx, requests, maxConcurrent)
}
