package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/cache"
)

type CacheRepository interface {
	IsCached(ctx context.Context, article Item) (bool, error)
	MarkAsProcessed(ctx context.Context, article Item) error
	Close() error
}

type cacheRepository struct {
	manager *cache.CloudStorageCache
}

func NewCacheRepository(manager *cache.CloudStorageCache) CacheRepository {
	return &cacheRepository{
		manager: manager,
	}
}

func (c *cacheRepository) IsCached(ctx context.Context, article Item) (bool, error) {
	// Convert repository.Item to rss.Item for cache compatibility
	rssItem := c.toRSSItem(article)
	return cache.IsCached(ctx, c.manager, rssItem)
}

func (c *cacheRepository) MarkAsProcessed(ctx context.Context, article Item) error {
	// Convert repository.Item to rss.Item for cache compatibility
	rssItem := c.toRSSItem(article)
	return cache.MarkAsProcessed(ctx, c.manager, rssItem)
}

func (c *cacheRepository) Close() error {
	return c.manager.Close()
}

// toRSSItem converts repository.Item to cache.RSSItem for cache compatibility
func (c *cacheRepository) toRSSItem(item Item) cache.RSSItem {
	return &item
}