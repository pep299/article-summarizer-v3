package mocks

import (
	"context"

	"github.com/pep299/article-summarizer-v3/internal/repository"
)

// Mock Gemini Repository
type MockGeminiRepo struct{}

func (m *MockGeminiRepo) SummarizeURL(ctx context.Context, url string) (*repository.SummarizeResponse, error) {
	return &repository.SummarizeResponse{Summary: "test summary"}, nil
}

func (m *MockGeminiRepo) SummarizeURLForOnDemand(ctx context.Context, url string) (*repository.SummarizeResponse, error) {
	return &repository.SummarizeResponse{Summary: "test summary"}, nil
}

func (m *MockGeminiRepo) SummarizeText(ctx context.Context, text string) (string, error) {
	return "test summary", nil
}

func (m *MockGeminiRepo) SummarizeComments(ctx context.Context, text string) (*repository.SummarizeResponse, error) {
	return &repository.SummarizeResponse{Summary: "test comment summary"}, nil
}

func (m *MockGeminiRepo) SummarizeOnDemand(ctx context.Context, url string) (*repository.SummarizeResponse, error) {
	return &repository.SummarizeResponse{Summary: "test summary"}, nil
}
