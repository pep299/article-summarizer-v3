package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/infrastructure"
)

type CacheRepository interface {
	IsCached(ctx context.Context, article Item) (bool, error)
	MarkAsProcessed(ctx context.Context, article Item) error
	Close() error
}

type cacheRepository struct {
	manager *infrastructure.CloudStorageCache
}

func NewCacheRepository(manager *infrastructure.CloudStorageCache) CacheRepository {
	return &cacheRepository{
		manager: manager,
	}
}

func (c *cacheRepository) IsCached(ctx context.Context, article Item) (bool, error) {
	// Convert repository.Item to rss.Item for cache compatibility
	rssItem := c.toRSSItem(article)
	return infrastructure.IsCached(ctx, c.manager, rssItem)
}

func (c *cacheRepository) MarkAsProcessed(ctx context.Context, article Item) error {
	// Convert repository.Item to rss.Item for cache compatibility
	rssItem := c.toRSSItem(article)
	return infrastructure.MarkAsProcessed(ctx, c.manager, rssItem)
}

func (c *cacheRepository) Close() error {
	return c.manager.Close()
}

// toRSSItem converts repository.Item to infrastructure.RSSItem for cache compatibility
func (c *cacheRepository) toRSSItem(item Item) infrastructure.RSSItem {
	return &item
}