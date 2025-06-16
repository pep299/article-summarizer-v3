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

type HatenaHandler struct {
	processor *article.HatenaProcessor
}

func NewHatenaHandler(
	rssRepo repository.RSSRepository,
	geminiRepo repository.GeminiRepository,
	slackRepo repository.SlackRepository,
	processedRepo repository.ProcessedArticleRepository,
	limiter limiter.ArticleLimiter,
) *HatenaHandler {
	return &HatenaHandler{
		processor: article.NewHatenaProcessor(rssRepo, geminiRepo, slackRepo, processedRepo, limiter),
	}
}

func (h *HatenaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	logger.Printf("Hatena feed processing request started")

	// Process Hatena feed
	if err := h.processor.Process(r.Context()); err != nil {
		logger.Printf("Error processing Hatena feed: %v", err)
		response.WriteInternalError(w, "Failed to process Hatena feed")
		return
	}

	logger.Printf("Hatena feed processing completed successfully")
	response.WriteSuccess(w, "Hatena feed processed successfully", nil)
}
