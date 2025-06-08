package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/model"
)

type RSSRepository interface {
	FetchFeed(ctx context.Context, feedName, url string) ([]model.Article, error)
}