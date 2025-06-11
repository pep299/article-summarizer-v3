package limiter

import "github.com/pep299/article-summarizer-v3/internal/repository"

// ArticleLimiter defines the interface for limiting articles before processing
type ArticleLimiter interface {
	Limit(articles []repository.Item) []repository.Item
}
