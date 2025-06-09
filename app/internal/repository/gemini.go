package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
)

type GeminiRepository interface {
	SummarizeURL(ctx context.Context, url string) (*gemini.SummarizeResponse, error)
}

type geminiRepository struct {
	client *gemini.Client
}

func NewGeminiRepository(client *gemini.Client) GeminiRepository {
	return &geminiRepository{
		client: client,
	}
}

func (g *geminiRepository) SummarizeURL(ctx context.Context, url string) (*gemini.SummarizeResponse, error) {
	return g.client.SummarizeURL(ctx, url)
}