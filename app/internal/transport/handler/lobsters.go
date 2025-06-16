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

type LobstersHandler struct {
	processor *article.LobstersProcessor
}

func NewLobstersHandler(
	rssRepo repository.RSSRepository,
	geminiRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
	processedRepo repository.ProcessedArticleRepository,
	limiter limiter.ArticleLimiter,
) *LobstersHandler {
	return &LobstersHandler{
		processor: article.NewLobstersProcessor(rssRepo, geminiRepo, slackRepo, processedRepo, limiter),
	}
}

func (h *LobstersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	logger.Printf("Lobsters feed processing request started")

	// Process Lobsters feed
	if err := h.processor.Process(r.Context()); err != nil {
		logger.Printf("Error processing Lobsters feed: %v", err)
		response.WriteInternalError(w, "Failed to process Lobsters feed")
		return
	}

	logger.Printf("Lobsters feed processing completed successfully")
	response.WriteSuccess(w, "Lobsters feed processed successfully", nil)
}
