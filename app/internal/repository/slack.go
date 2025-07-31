package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)


// Notification represents a unified notification structure
type Notification struct {
	Title        string
	Source       string // "reddit" | "hatena" | "lobsters" | "ondemand"
	URL          string
	Summary      string
	ContentChars int // Original content character count
}

type SlackRepository interface {
	Send(ctx context.Context, notification Notification) error
	SendOnDemandSummary(ctx context.Context, article Item, summary SummarizeResponse, targetChannel string) error
}

type slackRepository struct {
	botToken   string
	channel    string
	baseURL    string
	httpClient *http.Client
}

func NewSlackRepository(botToken, channel, baseURL string) SlackRepository {
	return &slackRepository{
		botToken: botToken,
		channel:  channel,
		baseURL:  baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}



func (s *slackRepository) sendMessage(ctx context.Context, message, channel string) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)

	type chatPostMessageRequest struct {
		Channel   string `json:"channel"`
		Text      string `json:"text"`
		Username  string `json:"username,omitempty"`
		IconEmoji string `json:"icon_emoji,omitempty"`
	}

	req := chatPostMessageRequest{
		Channel:   channel,
		Text:      message,
		Username:  "Article Summarizer",
		IconEmoji: ":robot_face:",
	}

	body, err := json.Marshal(req)
	if err != nil {
		logger.Printf("Error marshaling Slack request: %v", err)
		return fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		logger.Printf("Error creating Slack HTTP request: %v", err)
		return fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.botToken))

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		logger.Printf("Error sending request to Slack API: %v request_body=%s request_headers=%v\nStack:\n%s", err, string(body), httpReq.Header, debug.Stack())
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		responseBody, _ := io.ReadAll(resp.Body)
		logger.Printf("Slack API request failed channel=%s status_code=%d request_body=%s request_headers=%v response_headers=%v response_body=%s\nStack:\n%s",
			channel, resp.StatusCode, string(body), httpReq.Header, resp.Header, string(responseBody), debug.Stack())
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (s *slackRepository) SendOnDemandSummary(ctx context.Context, article Item, summary SummarizeResponse, targetChannel string) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	// Use default channel if targetChannel is empty
	channel := targetChannel
	if channel == "" {
		channel = s.channel
	}

	logger.Printf("On-demand Slack notification started url=%s channel=%s", article.Link, channel)
	message := s.formatOnDemandMessage(article, summary)
	if err := s.sendMessage(ctx, message, channel); err != nil {
		logger.Printf("Error sending on-demand summary to Slack: %v", err)
		return err
	}

	duration := time.Since(start)
	logger.Printf("On-demand Slack notification completed url=%s channel=%s duration_ms=%d",
		article.Link, channel, duration.Milliseconds())
	return nil
}

func (s *slackRepository) formatOnDemandMessage(article Item, summary SummarizeResponse) string {
	timestamp := time.Now().In(time.FixedZone("JST", 9*3600)).Format("2006-01-02 15:04:05")
	title := article.Title
	if title == "" {
		title = "タイトル取得中..."
	}

	return fmt.Sprintf(`🔗 *オンデマンド要約リクエスト完了*

*%s*
🔗 URL: %s
📊 コンテンツ文字数: %d文字

%s

📝 要約方法: オンデマンドAPI
⏰ 処理時刻: %s`,
		title,
		article.Link,
		summary.ContentChars,
		summary.Summary,
		timestamp)
}

// Send sends a unified notification
func (s *slackRepository) Send(ctx context.Context, notification Notification) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Slack notification started title=%s source=%s channel=%s",
		notification.Title, notification.Source, s.channel)

	message := s.formatNotification(notification)
	if err := s.sendMessage(ctx, message, s.channel); err != nil {
		logger.Printf("Error sending notification to Slack: %v", err)
		return err
	}

	duration := time.Since(start)
	logger.Printf("Slack notification completed title=%s source=%s channel=%s duration_ms=%d",
		notification.Title, notification.Source, s.channel, duration.Milliseconds())
	return nil
}


func (s *slackRepository) formatNotification(notification Notification) string {
	timestamp := time.Now().In(time.FixedZone("JST", 9*3600)).Format("2006-01-02 15:04:05")

	return fmt.Sprintf(`*%s*
📰 ソース: %s
🔗 URL: %s
📊 コンテンツ文字数: %d文字

%s

⏰ 処理時刻: %s`,
		notification.Title,
		notification.Source,
		notification.URL,
		notification.ContentChars,
		notification.Summary,
		timestamp)
}

func (s *slackRepository) formatArticleMessage(article Item, summary SummarizeResponse) string {
	timestamp := time.Now().In(time.FixedZone("JST", 9*3600)).Format("2006-01-02 15:04:05")

	return fmt.Sprintf(`📄 *記事要約*

*%s*
📰 ソース: %s
🔗 URL: %s

%s

⏰ 処理時刻: %s`,
		article.Title,
		article.Source,
		article.Link,
		summary.Summary,
		timestamp)
}

