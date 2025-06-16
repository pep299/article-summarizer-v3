package handler

import (
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/service/article"
	"github.com/pep299/article-summarizer-v3/internal/service/limiter"
	"github.com/pep299/article-summarizer-v3/internal/transport/response"
)

type RedditHandler struct {
	processor *article.RedditProcessor
}

func NewRedditHandler(
	rssRepo repository.RSSRepository,
	geminiRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
	processedRepo repository.ProcessedArticleRepository,
	limiter limiter.ArticleLimiter,
) *RedditHandler {
	return &RedditHandler{
		processor: article.NewRedditProcessor(rssRepo, geminiRepo, slackRepo, processedRepo, limiter),
	}
}

func (h *RedditHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	logger.Printf("Reddit feed processing request started")

	// Process Reddit feed
	if err := h.processor.Process(r.Context()); err != nil {
		logger.Printf("Error processing Reddit feed: %v", err)
		response.WriteInternalError(w, "Failed to process Reddit feed")
		return
	}

	logger.Printf("Reddit feed processing completed successfully")
	response.WriteSuccess(w, "Reddit feed processed successfully", nil)
}
