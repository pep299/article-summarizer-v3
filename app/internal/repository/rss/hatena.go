package rss

import (
	"context"
	"encoding/xml"
	"fmt"
	"os"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

type HatenaRSSRepository struct {
	rssRepo repository.RSSRepository
}

func NewHatenaRSSRepository(rssRepo repository.RSSRepository) *HatenaRSSRepository {
	return &HatenaRSSRepository{
		rssRepo: rssRepo,
	}
}

func (h *HatenaRSSRepository) FetchArticles(ctx context.Context) ([]repository.Item, error) {
	url := "https://b.hatena.ne.jp/hotentry/it.rss"
	// テスト用URLオーバーライド
	if testURL := os.Getenv("HATENA_RSS_URL"); testURL != "" {
		url = testURL
	}

	headers := map[string]string{
		"User-Agent": "Article Summarizer Bot/1.0 (Hatena)",
		"Accept":     "application/rdf+xml, application/xml, text/xml",
	}

	xmlContent, err := h.rssRepo.FetchFeedXML(ctx, url, headers)
	if err != nil {
		return nil, fmt.Errorf("fetching Hatena RSS: %w", err)
	}

	return h.parseFeed(xmlContent)
}

func (h *HatenaRSSRepository) FetchComments(ctx context.Context, commentURL string) (*Comments, error) {
	// Hatena doesn't support comment fetching yet
	return nil, fmt.Errorf("comment fetching not implemented for Hatena")
}

func (h *HatenaRSSRepository) parseFeed(xmlContent string) ([]repository.Item, error) {
	// Hatena uses RDF format
	var rdf struct {
		Items []struct {
			Title       string `xml:"title"`
			Link        string `xml:"link"`
			Description string `xml:"description"`
			PubDate     string `xml:"pubDate"`
			GUID        string `xml:"guid"`
		} `xml:"item"`
	}

	if err := xml.Unmarshal([]byte(xmlContent), &rdf); err != nil {
		return nil, fmt.Errorf("failed to parse Hatena RDF format: %w", err)
	}

	var items []repository.Item
	for _, item := range rdf.Items {
		parsedDate, _ := h.parseDate(item.PubDate)

		items = append(items, repository.Item{
			Title:       item.Title,
			Link:        item.Link,
			Description: item.Description,
			PubDate:     item.PubDate,
			GUID:        item.GUID,
			ParsedDate:  parsedDate,
			Source:      "hatena",
			CommentURL:  "", // Hatena doesn't have separate comment URLs
		})
	}

	return items, nil
}

func (h *HatenaRSSRepository) parseDate(dateStr string) (time.Time, error) {
	// Hatena uses standard RFC formats
	formats := []string{
		time.RFC3339, // Most common for Hatena
		time.RFC1123Z,
		time.RFC1123,
		"2006-01-02T15:04:05Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse Hatena date: %s", dateStr)
}
