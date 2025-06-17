package article

import (
	"context"
	"fmt"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// filterUnprocessedArticles filters out already processed articles
func filterUnprocessedArticles(ctx context.Context, processedRepo repository.ProcessedArticleRepository, articles []repository.Item) ([]repository.Item, error) {
	// Load index once at the beginning
	index, err := processedRepo.LoadIndex(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading index: %w", err)
	}

	var unprocessed []repository.Item
	for _, article := range articles {
		key := processedRepo.GenerateKey(article)
		processed := processedRepo.IsProcessed(key, index)

		if !processed {
			unprocessed = append(unprocessed, article)
		}
	}

	return unprocessed, nil
}
