package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

// Client handles Slack notifications
type Client struct {
	webhookURL string
	channel    string
	httpClient *http.Client
}

// NewClient creates a new Slack client
func NewClient(webhookURL, channel string) *Client {
	return &Client{
		webhookURL: webhookURL,
		channel:    channel,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Message represents a Slack message
type Message struct {
	Channel     string       `json:"channel,omitempty"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Text        string       `json:"text,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	Blocks      []Block      `json:"blocks,omitempty"`
}

// Attachment represents a Slack attachment
type Attachment struct {
	Color      string  `json:"color,omitempty"`
	Title      string  `json:"title,omitempty"`
	TitleLink  string  `json:"title_link,omitempty"`
	Text       string  `json:"text,omitempty"`
	Fields     []Field `json:"fields,omitempty"`
	Footer     string  `json:"footer,omitempty"`
	Timestamp  int64   `json:"ts,omitempty"`
	MarkdownIn []string `json:"mrkdwn_in,omitempty"`
}

// Field represents a Slack attachment field
type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

// Block represents a Slack block
type Block struct {
	Type string      `json:"type"`
	Text *TextObject `json:"text,omitempty"`
}

// TextObject represents a Slack text object
type TextObject struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ArticleSummary represents an article summary for notification
type ArticleSummary struct {
	RSS     rss.Item
	Summary gemini.SummarizeResponse
}

// SendMessage sends a message to Slack
func (c *Client) SendMessage(ctx context.Context, message Message) error {
	if c.channel != "" && message.Channel == "" {
		message.Channel = c.channel
	}

	body, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned status %d", resp.StatusCode)
	}

	return nil
}

// SendArticleSummary sends an article summary to Slack
func (c *Client) SendArticleSummary(ctx context.Context, summary ArticleSummary) error {
	message := c.buildArticleMessage(summary)
	return c.SendMessage(ctx, message)
}

// SendMultipleSummaries sends multiple article summaries to Slack
func (c *Client) SendMultipleSummaries(ctx context.Context, summaries []ArticleSummary) error {
	if len(summaries) == 0 {
		return nil
	}

	if len(summaries) == 1 {
		return c.SendArticleSummary(ctx, summaries[0])
	}

	// For multiple summaries, create a digest message
	message := c.buildDigestMessage(summaries)
	return c.SendMessage(ctx, message)
}

// buildArticleMessage creates a Slack message for a single article
func (c *Client) buildArticleMessage(summary ArticleSummary) Message {
	color := c.getCategoryColor(summary.Summary.Category)
	
	attachment := Attachment{
		Color:     color,
		Title:     summary.RSS.Title,
		TitleLink: summary.RSS.Link,
		Text:      summary.Summary.Summary,
		Fields: []Field{
			{
				Title: "カテゴリ",
				Value: summary.Summary.Category,
				Short: true,
			},
			{
				Title: "信頼度",
				Value: fmt.Sprintf("%.1f%%", summary.Summary.Confidence*100),
				Short: true,
			},
		},
		Footer:     "Article Summarizer v3",
		Timestamp:  summary.Summary.ProcessedAt.Unix(),
		MarkdownIn: []string{"text", "fields"},
	}

	// Add key points if available
	if len(summary.Summary.KeyPoints) > 0 {
		keyPointsText := "• " + strings.Join(summary.Summary.KeyPoints, "\n• ")
		attachment.Fields = append(attachment.Fields, Field{
			Title: "重要なポイント",
			Value: keyPointsText,
			Short: false,
		})
	}

	return Message{
		Username:  "記事要約Bot",
		IconEmoji: ":newspaper:",
		Text:      "新しい記事の要約をお届けします :newspaper:",
		Attachments: []Attachment{attachment},
	}
}

// buildDigestMessage creates a Slack message for multiple articles
func (c *Client) buildDigestMessage(summaries []ArticleSummary) Message {
	text := fmt.Sprintf(":newspaper: *記事要約ダイジェスト* - %d件の記事", len(summaries))
	
	var attachments []Attachment
	
	for i, summary := range summaries {
		if i >= 10 { // Limit to 10 articles to avoid message size limits
			remaining := len(summaries) - i
			attachments = append(attachments, Attachment{
				Color: "#cccccc",
				Text:  fmt.Sprintf("他 %d件の記事があります...", remaining),
			})
			break
		}

		color := c.getCategoryColor(summary.Summary.Category)
		
		attachment := Attachment{
			Color:     color,
			Title:     summary.RSS.Title,
			TitleLink: summary.RSS.Link,
			Text:      summary.Summary.Summary,
			Fields: []Field{
				{
					Title: "カテゴリ",
					Value: summary.Summary.Category,
					Short: true,
				},
			},
			MarkdownIn: []string{"text"},
		}
		
		attachments = append(attachments, attachment)
	}

	return Message{
		Username:    "記事要約Bot",
		IconEmoji:   ":newspaper:",
		Text:        text,
		Attachments: attachments,
	}
}

// getCategoryColor returns a color based on the article category
func (c *Client) getCategoryColor(category string) string {
	categoryLower := strings.ToLower(category)
	
	switch {
	case strings.Contains(categoryLower, "技術") || strings.Contains(categoryLower, "tech"):
		return "#36a64f" // Green
	case strings.Contains(categoryLower, "ビジネス") || strings.Contains(categoryLower, "business"):
		return "#2eb886" // Blue-green
	case strings.Contains(categoryLower, "ニュース") || strings.Contains(categoryLower, "news"):
		return "#ff6b6b" // Red
	case strings.Contains(categoryLower, "ai") || strings.Contains(categoryLower, "人工知能"):
		return "#9c88ff" // Purple
	case strings.Contains(categoryLower, "開発") || strings.Contains(categoryLower, "dev"):
		return "#ffa500" // Orange
	default:
		return "#cccccc" // Gray
	}
}

// SendSimpleMessage sends a simple text message to Slack
func (c *Client) SendSimpleMessage(ctx context.Context, text string) error {
	message := Message{
		Username:  "記事要約Bot",
		IconEmoji: ":robot_face:",
		Text:      text,
	}
	
	return c.SendMessage(ctx, message)
}

// SendErrorMessage sends an error message to Slack
func (c *Client) SendErrorMessage(ctx context.Context, errorMsg string) error {
	attachment := Attachment{
		Color: "#ff0000", // Red
		Title: "エラーが発生しました",
		Text:  errorMsg,
		Footer: "Article Summarizer v3",
		Timestamp: time.Now().Unix(),
	}

	message := Message{
		Username:  "記事要約Bot",
		IconEmoji: ":warning:",
		Text:      ":warning: システムエラーが発生しました",
		Attachments: []Attachment{attachment},
	}
	
	return c.SendMessage(ctx, message)
}

// SendStatusMessage sends a status update message to Slack
func (c *Client) SendStatusMessage(ctx context.Context, status string, details map[string]interface{}) error {
	var fields []Field
	
	for key, value := range details {
		fields = append(fields, Field{
			Title: key,
			Value: fmt.Sprintf("%v", value),
			Short: true,
		})
	}

	attachment := Attachment{
		Color:     "#36a64f", // Green
		Title:     "システム状況",
		Text:      status,
		Fields:    fields,
		Footer:    "Article Summarizer v3",
		Timestamp: time.Now().Unix(),
	}

	message := Message{
		Username:  "記事要約Bot",
		IconEmoji: ":information_source:",
		Text:      ":information_source: システム状況をお知らせします",
		Attachments: []Attachment{attachment},
	}
	
	return c.SendMessage(ctx, message)
}
