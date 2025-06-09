package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/rss"
)

type RSSRepository interface {
	FetchFeed(ctx context.Context, feedName, url string) ([]rss.Item, error)
}

type rssRepository struct {
	client *rss.Client
}

func NewRSSRepository(client *rss.Client) RSSRepository {
	return &rssRepository{
		client: client,
	}
}

func (r *rssRepository) FetchFeed(ctx context.Context, feedName, url string) ([]rss.Item, error) {
	return r.client.FetchFeed(ctx, feedName, url)
}