package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Slack Repository
type MockSlackRepo struct {
	SentNotifications []repository.Notification
}

func (m *MockSlackRepo) Send(ctx context.Context, notification repository.Notification) error {
	m.SentNotifications = append(m.SentNotifications, notification)
	return nil
}

func (m *MockSlackRepo) SendOnDemandSummary(ctx context.Context, article repository.Item, summary repository.SummarizeResponse, targetChannel string) error {
	return nil
}
