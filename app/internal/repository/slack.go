package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/model"
)

type SlackRepository interface {
	SendArticleSummary(ctx context.Context, summary model.ArticleSummary) error
}