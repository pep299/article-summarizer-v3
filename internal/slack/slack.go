package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

// Client handles Slack notifications
type Client struct {
	botToken   string
	channel    string
	httpClient *http.Client
}

// NewClient creates a new Slack client
func NewClient(botToken, channel string) *Client {
	return &Client{
		botToken:   botToken,
		channel:    channel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ArticleSummary represents an article summary for notification
type ArticleSummary struct {
	RSS     rss.Item
	Summary gemini.SummarizeResponse
}

// ChatPostMessageRequest represents a Slack chat.postMessage request
type ChatPostMessageRequest struct {
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	Username  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

// SendArticleSummary sends an article summary to Slack (RSS mode)
func (c *Client) SendArticleSummary(ctx context.Context, summary ArticleSummary) error {
	message := c.formatRSSMessage(summary)
	return c.sendMessage(ctx, message, c.channel)
}

// SendOnDemandSummary sends an on-demand summary to Slack with specified channel
func (c *Client) SendOnDemandSummary(ctx context.Context, article rss.Item, summary gemini.SummarizeResponse, targetChannel string) error {
	message := c.formatOnDemandMessage(article, summary)
	return c.sendMessage(ctx, message, targetChannel)
}

// formatRSSMessage creates a Slack message for RSS articles (same as v1)
func (c *Client) formatRSSMessage(summary ArticleSummary) string {
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

// formatOnDemandMessage creates a Slack message for on-demand requests (same as v1)
func (c *Client) formatOnDemandMessage(article rss.Item, summary gemini.SummarizeResponse) string {
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

// sendMessage sends a message to the specified Slack channel
func (c *Client) sendMessage(ctx context.Context, text string, channel string) error {
	req := ChatPostMessageRequest{
		Channel:   channel,
		Text:      text,
		Username:  "Article Summarizer",
		IconEmoji: ":robot_face:",
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://slack.com/api/chat.postMessage", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.botToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	var slackResp struct {
		OK    bool   `json:"ok"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&slackResp); err != nil {
		return fmt.Errorf("decoding response: %w", err)
	}

	if !slackResp.OK {
		return fmt.Errorf("slack API error: %s", slackResp.Error)
	}

	return nil
}

// SendSimpleMessage sends a simple text message to Slack
func (c *Client) SendSimpleMessage(ctx context.Context, text string) error {
	return c.sendMessage(ctx, text, c.channel)
}
