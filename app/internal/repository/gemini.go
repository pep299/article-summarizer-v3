package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"runtime/debug"
	"strings"
	"time"
)

// SummarizeResponse represents a summarization response
type SummarizeResponse struct {
	Summary     string    `json:"summary"`
	ProcessedAt time.Time `json:"processed_at"`
}

type GeminiRepository interface {
	SummarizeURL(ctx context.Context, url string) (*SummarizeResponse, error)
	SummarizeURLForOnDemand(ctx context.Context, url string) (*SummarizeResponse, error)
}

type geminiRepository struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

func NewGeminiRepository(apiKey, model string) GeminiRepository {
	return &geminiRepository{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://generativelanguage.googleapis.com/v1beta/models",
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (g *geminiRepository) SummarizeURL(ctx context.Context, url string) (*SummarizeResponse, error) {
	// Fetch HTML content
	htmlContent, err := g.fetchHTML(ctx, url)
	if err != nil {
		log.Printf("Error fetching HTML from URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return nil, fmt.Errorf("fetching HTML: %w", err)
	}

	// Extract text from HTML
	textContent := g.extractTextFromHTML(htmlContent)
	if textContent == "" {
		return nil, fmt.Errorf("no text content found")
	}

	// Create prompt for RSS mode (shorter summary for team sharing)
	prompt := g.buildRSSPrompt(textContent)

	// Call Gemini API
	summary, err := g.callGeminiAPI(ctx, prompt)
	if err != nil {
		log.Printf("Error calling Gemini API for URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return nil, err
	}

	return &SummarizeResponse{
		Summary:     summary,
		ProcessedAt: time.Now(),
	}, nil
}

func (g *geminiRepository) fetchHTML(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Article Summarizer Bot/1.0)")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		log.Printf("Error making HTTP request to URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body from URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return "", fmt.Errorf("reading response body: %w", err)
	}

	return string(body), nil
}

func (g *geminiRepository) extractTextFromHTML(html string) string {
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

func (g *geminiRepository) buildRSSPrompt(textContent string) string {
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

// Gemini API types
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

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

func (g *geminiRepository) callGeminiAPI(ctx context.Context, prompt string) (string, error) {

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

	url := fmt.Sprintf("%s/%s:generateContent?key=%s", g.baseURL, g.model, g.apiKey)

	body, err := json.Marshal(geminiReq)
	if err != nil {
		log.Printf("Error marshaling Gemini request: %v\nStack:\n%s", err, debug.Stack())
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		log.Printf("Error creating Gemini API request: %v\nStack:\n%s", err, debug.Stack())
		return "", fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		log.Printf("Error sending request to Gemini API: %v\nStack:\n%s", err, debug.Stack())
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		log.Printf("Error decoding Gemini API response: %v\nStack:\n%s", err, debug.Stack())
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *geminiRepository) SummarizeURLForOnDemand(ctx context.Context, url string) (*SummarizeResponse, error) {
	// Fetch HTML content
	htmlContent, err := g.fetchHTML(ctx, url)
	if err != nil {
		log.Printf("Error fetching HTML for on-demand from URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return nil, fmt.Errorf("fetching HTML: %w", err)
	}

	// Extract text from HTML
	textContent := g.extractTextFromHTML(htmlContent)
	if textContent == "" {
		return nil, fmt.Errorf("no text content found")
	}

	// Create prompt for on-demand mode (longer summary for individual requests)
	prompt := g.buildOnDemandPrompt(textContent)

	// Call Gemini API
	summary, err := g.callGeminiAPI(ctx, prompt)
	if err != nil {
		log.Printf("Error calling Gemini API for on-demand URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return nil, err
	}

	return &SummarizeResponse{
		Summary:     summary,
		ProcessedAt: time.Now(),
	}, nil
}

func (g *geminiRepository) buildOnDemandPrompt(textContent string) string {
	// Limit content to 10KB
	if len(textContent) > 10000 {
		textContent = textContent[:10000]
	}

	return fmt.Sprintf(`以下のテキストを、個人のリクエストに応じて詳細に要約してください。800-1200文字程度の詳細な要約を作成してください。

**重要な制約:**
- 推測や創作は一切せず、実際に記載されている内容のみを要約してください
- 記載されていない情報は追加しないでください

オンデマンド要約として、以下の構造で詳細に出力してください：
- 📝 **概要:** 実際の内容を4-6行で詳細に
- 🎯 **対象者・課題:** 記載されている対象や課題を詳しく
- 💡 **解決策・効果:** 明記されている解決策や効果を具体的に
- 🔍 **技術的詳細:** 技術的な内容があれば詳細に
- 📊 **結果・データ:** 具体的な数値や結果があれば詳しく

テキスト内容:
%s`, textContent)
}
