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

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

// SummarizeResponse represents a summarization response
type SummarizeResponse struct {
	Summary     string    `json:"summary"`
	ProcessedAt time.Time `json:"processed_at"`
}

type GeminiRepository interface {
	SummarizeURL(ctx context.Context, url string) (*SummarizeResponse, error)
	SummarizeURLForOnDemand(ctx context.Context, url string) (*SummarizeResponse, error)
	SummarizeText(ctx context.Context, text string) (string, error)
	SummarizeComments(ctx context.Context, commentsText string) (string, error)
}

type geminiRepository struct {
	apiKey     string
	model      string
	httpClient *http.Client
	baseURL    string
}

func NewGeminiRepository(apiKey, model, baseURL string) GeminiRepository {
	return &geminiRepository{
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (g *geminiRepository) SummarizeURL(ctx context.Context, url string) (*SummarizeResponse, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("HTML fetch started url=%s", url)
	// Fetch HTML content
	htmlContent, err := g.fetchHTML(ctx, url)
	if err != nil {
		logger.Printf("Error fetching HTML from URL %s: %v", url, err)
		return nil, fmt.Errorf("fetching HTML: %w", err)
	}

	fetchDuration := time.Since(start)
	logger.Printf("HTML fetch completed url=%s content_length=%d duration_ms=%d", url, len(htmlContent), fetchDuration.Milliseconds())

	// Extract text from HTML
	textContent := g.extractTextFromHTML(htmlContent)
	if textContent == "" {
		logger.Printf("No text content found url=%s", url)
		return nil, fmt.Errorf("no text content found")
	}

	logger.Printf("Text extraction completed url=%s text_length=%d", url, len(textContent))

	// Create prompt for RSS mode (shorter summary for team sharing)
	prompt := g.buildRSSPrompt(textContent)

	// Call Gemini API
	geminiStart := time.Now()
	logger.Printf("Gemini API call started url=%s", url)
	summary, err := g.callGeminiAPI(ctx, prompt)
	if err != nil {
		logger.Printf("Error calling Gemini API for URL %s: %v", url, err)
		return nil, err
	}

	geminiDuration := time.Since(geminiStart)
	totalDuration := time.Since(start)
	logger.Printf("Gemini API completed url=%s summary_length=%d gemini_duration_ms=%d total_duration_ms=%d",
		url, len(summary), geminiDuration.Milliseconds(), totalDuration.Milliseconds())

	return &SummarizeResponse{
		Summary:     summary,
		ProcessedAt: time.Now(),
	}, nil
}

func (g *geminiRepository) fetchHTML(ctx context.Context, url string) (string, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logger.Printf("Error creating HTTP request url=%s error=%v", url, err)
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Article Summarizer Bot/1.0)")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		logger.Printf("Error making HTTP request to URL %s: %v\nStack:\n%s", url, err, debug.Stack())
		return "", fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read response body for error details
		responseBody, _ := io.ReadAll(resp.Body)

		// Log detailed error information
		logger.Printf("HTTP request failed url=%s status_code=%d request_headers=%v response_headers=%v response_body=%s\nStack:\n%s",
			url, resp.StatusCode, req.Header, resp.Header, string(responseBody), debug.Stack())
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Printf("Error reading response body from URL %s: %v", url, err)
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
	logger := log.New(funcframework.LogWriter(ctx), "", 0)

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
		logger.Printf("Error marshaling Gemini request: %v", err)
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		logger.Printf("Error creating Gemini API request: %v", err)
		return "", fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		logger.Printf("Error sending request to Gemini API: %v\nStack:\n%s", err, debug.Stack())
		return "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.Printf("Gemini API request failed status_code=%d response=%s\nStack:\n%s", resp.StatusCode, string(bodyBytes), debug.Stack())
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		logger.Printf("Error decoding Gemini API response: %v", err)
		return "", fmt.Errorf("decoding response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		logger.Printf("No content in Gemini API response")
		return "", fmt.Errorf("no content in response")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func (g *geminiRepository) SummarizeURLForOnDemand(ctx context.Context, url string) (*SummarizeResponse, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("On-demand HTML fetch started url=%s", url)
	// Fetch HTML content
	htmlContent, err := g.fetchHTML(ctx, url)
	if err != nil {
		logger.Printf("Error fetching HTML for on-demand from URL %s: %v", url, err)
		return nil, fmt.Errorf("fetching HTML: %w", err)
	}

	fetchDuration := time.Since(start)
	logger.Printf("On-demand HTML fetch completed url=%s content_length=%d duration_ms=%d", url, len(htmlContent), fetchDuration.Milliseconds())

	// Extract text from HTML
	textContent := g.extractTextFromHTML(htmlContent)
	if textContent == "" {
		logger.Printf("No text content found for on-demand url=%s", url)
		return nil, fmt.Errorf("no text content found")
	}

	logger.Printf("On-demand text extraction completed url=%s text_length=%d", url, len(textContent))

	// Create prompt for on-demand mode (longer summary for individual requests)
	prompt := g.buildOnDemandPrompt(textContent)

	// Call Gemini API
	geminiStart := time.Now()
	logger.Printf("On-demand Gemini API call started url=%s", url)
	summary, err := g.callGeminiAPI(ctx, prompt)
	if err != nil {
		logger.Printf("Error calling Gemini API for on-demand URL %s: %v", url, err)
		return nil, err
	}

	geminiDuration := time.Since(geminiStart)
	totalDuration := time.Since(start)
	logger.Printf("On-demand Gemini API completed url=%s summary_length=%d gemini_duration_ms=%d total_duration_ms=%d",
		url, len(summary), geminiDuration.Milliseconds(), totalDuration.Milliseconds())

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

// SummarizeText summarizes the provided text content directly
func (g *geminiRepository) SummarizeText(ctx context.Context, text string) (string, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	
	// Build prompt for text summarization (using detailed on-demand format)
	prompt := g.buildOnDemandPrompt(text)
	
	logger.Printf("Text summarization started text_length=%d", len(text))
	
	// Call Gemini API
	geminiStart := time.Now()
	summary, err := g.callGeminiAPI(ctx, prompt)
	if err != nil {
		logger.Printf("Error calling Gemini API for text summarization: %v", err)
		return "", fmt.Errorf("calling Gemini API: %w", err)
	}
	
	geminiDuration := time.Since(geminiStart)
	logger.Printf("Text summarization completed summary_length=%d duration_ms=%d", 
		len(summary), geminiDuration.Milliseconds())
	
	return summary, nil
}

// SummarizeComments summarizes comments/discussions using specialized prompt
func (g *geminiRepository) SummarizeComments(ctx context.Context, commentsText string) (string, error) {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	
	// Build specialized prompt for comments
	prompt := g.buildCommentsPrompt(commentsText)
	
	logger.Printf("Comments summarization started text_length=%d", len(commentsText))
	
	// Call Gemini API
	geminiStart := time.Now()
	summary, err := g.callGeminiAPI(ctx, prompt)
	if err != nil {
		logger.Printf("Error calling Gemini API for comments summarization: %v", err)
		return "", fmt.Errorf("calling Gemini API: %w", err)
	}
	
	geminiDuration := time.Since(geminiStart)
	logger.Printf("Comments summarization completed summary_length=%d duration_ms=%d", 
		len(summary), geminiDuration.Milliseconds())
	
	return summary, nil
}

// buildCommentsPrompt creates specialized prompt for comments/discussions
func (g *geminiRepository) buildCommentsPrompt(commentsText string) string {
	// Limit content to 10KB for better focus and 1000-char summary
	if len(commentsText) > 10000 {
		commentsText = commentsText[:10000]
	}

	return fmt.Sprintf(`以下はオンラインディスカッション・コメントのテキストです。コミュニティの議論内容を分析し、1000文字以内で簡潔に要約してください。

**重要な制約:**
- 推測や創作は一切せず、実際のコメント内容のみを要約してください
- コメントに記載されていない情報は追加しないでください
- スコアの高いコメントや建設的な議論を重視してください
- 1000文字以内という制限を厳守してください

コメント要約として、以下の構造で簡潔に出力してください：
- 📝 **概要:** 議論の主なトピック（2-3行）
- 💡 **主要な解決策:** 提案された解決策や代替案（2-3行）
- 🔍 **技術的ポイント:** 重要な技術的議論（2-3行）
- 🗣️ **コミュニティの反応:** 注目すべき意見や傾向（1-2行）

コメント内容:
%s`, commentsText)
}
