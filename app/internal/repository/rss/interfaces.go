package rss

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Comments represents processed comment data
type Comments struct {
	Text string
}

// FeedRepository defines the interface for RSS feed data retrieval
type FeedRepository interface {
	FetchArticles(ctx context.Context) ([]repository.Item, error)
	FetchComments(ctx context.Context, commentURL string) (*Comments, error)
}
