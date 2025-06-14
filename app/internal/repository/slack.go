package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
)

// ArticleSummary represents an article summary for notification
type ArticleSummary struct {
	RSS     Item
	Summary SummarizeResponse
}

type SlackRepository interface {
	SendArticleSummary(ctx context.Context, summary ArticleSummary) error
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

func (s *slackRepository) SendArticleSummary(ctx context.Context, summary ArticleSummary) error {
	logger := log.New(funcframework.LogWriter(ctx), "", 0)
	start := time.Now()

	logger.Printf("Slack notification started title=%s channel=%s", summary.RSS.Title, s.channel)
	message := s.formatRSSMessage(summary)
	if err := s.sendMessage(ctx, message, s.channel); err != nil {
		logger.Printf("Error sending RSS article summary to Slack: %v", err)
		return err
	}

	duration := time.Since(start)
	logger.Printf("Slack notification completed title=%s channel=%s duration_ms=%d",
		summary.RSS.Title, s.channel, duration.Milliseconds())
	return nil
}

func (s *slackRepository) formatRSSMessage(summary ArticleSummary) string {
	timestamp := time.Now().In(time.FixedZone("JST", 9*3600)).Format("2006-01-02 15:04:05")

	return fmt.Sprintf(`🆕 *新しい記事を要約しました*

*%s*
📰 ソース: %s
🔗 URL: %s

%s

⏰ 処理時刻: %s`,
		summary.RSS.Title,
		summary.RSS.Source,
		summary.RSS.Link,
		summary.Summary.Summary,
		timestamp)
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
		logger.Printf("Error sending request to Slack API: %v\nStack:\n%s", err, debug.Stack())
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logger.Printf("Slack API request failed channel=%s status_code=%d\nStack:\n%s", channel, resp.StatusCode, debug.Stack())
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

%s

📝 要約方法: オンデマンドAPI
⏰ 処理時刻: %s`,
		title,
		article.Link,
		summary.Summary,
		timestamp)
}
