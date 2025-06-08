package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/model"
)

type GeminiRepository interface {
	SummarizeURL(ctx context.Context, url string) (*model.Summary, error)
}