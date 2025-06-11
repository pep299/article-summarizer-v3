package limiter

import "github.com/pep299/article-summarizer-v3/internal/repository"

// ProductionArticleLimiter returns all articles without any limiting
type ProductionArticleLimiter struct{}

func NewProductionArticleLimiter() *ProductionArticleLimiter {
	return &ProductionArticleLimiter{}
}

func (l *ProductionArticleLimiter) Limit(articles []repository.Item) []repository.Item {
	return articles
}
