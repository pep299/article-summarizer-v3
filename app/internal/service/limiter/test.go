package limiter

import (
	"log"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// TestArticleLimiter limits articles to 1 for testing purposes
type TestArticleLimiter struct{}

func NewTestArticleLimiter() *TestArticleLimiter {
	return &TestArticleLimiter{}
}

func (l *TestArticleLimiter) Limit(articles []repository.Item) []repository.Item {
	if len(articles) == 0 {
		return articles
	}

	if len(articles) > 1 {
		log.Printf("テスト用制限により 1件に制限 (元: %d件)", len(articles))
		return articles[:1]
	}

	return articles
}
