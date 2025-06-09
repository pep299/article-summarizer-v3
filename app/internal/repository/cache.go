package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

type CacheRepository interface {
	IsCached(ctx context.Context, article rss.Item) (bool, error)
	MarkAsProcessed(ctx context.Context, article rss.Item) error
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

func (c *cacheRepository) IsCached(ctx context.Context, article rss.Item) (bool, error) {
	return cache.IsCached(ctx, c.manager, article)
}

func (c *cacheRepository) MarkAsProcessed(ctx context.Context, article rss.Item) error {
	return cache.MarkAsProcessed(ctx, c.manager, article)
}

func (c *cacheRepository) Close() error {
	return c.manager.Close()
}