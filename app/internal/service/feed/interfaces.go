package feed

import (
	"time"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// FeedStrategy defines the interface for feed-specific processing
type FeedStrategy interface {
	GetConfig() FeedConfig
	FilterItems(items []repository.Item) []repository.Item
	ParseFeed(xmlContent string) ([]repository.Item, error)
	GetRequestHeaders() map[string]string
	ParseDate(dateStr string) (time.Time, error)
}

type FeedConfig struct {
	Name        string
	URL         string
	DisplayName string
}

