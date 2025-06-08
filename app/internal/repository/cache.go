package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/model"
)

type CacheRepository interface {
	IsCached(ctx context.Context, article model.Article) (bool, error)
	MarkAsProcessed(ctx context.Context, article model.Article) error
	Close() error
}