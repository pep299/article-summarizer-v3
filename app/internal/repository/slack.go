package repository

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/slack"
)

type SlackRepository interface {
	SendArticleSummary(ctx context.Context, summary slack.ArticleSummary) error
}

type slackRepository struct {
	client *slack.Client
}

func NewSlackRepository(client *slack.Client) SlackRepository {
	return &slackRepository{
		client: client,
	}
}

func (s *slackRepository) SendArticleSummary(ctx context.Context, summary slack.ArticleSummary) error {
	return s.client.SendArticleSummary(ctx, summary)
}