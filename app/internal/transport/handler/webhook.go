package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	"github.com/pep299/article-summarizer-v3/internal/service"
	"github.com/pep299/article-summarizer-v3/internal/transport/response"
)

type Webhook struct {
	urlService *service.URL
}

func NewWebhook(urlService *service.URL) *Webhook {
	return &Webhook{
		urlService: urlService,
	}
}

type webhookRequest struct {
	URL string `json:"url"`
}

func (h *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Printf("Invalid JSON in webhook request: %v\nStack:\n%s", err, debug.Stack())
		response.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.URL == "" {
		logger.Printf("Missing URL in webhook request")
		response.WriteBadRequest(w, "URL is required")
		return
	}

	logger.Printf("Webhook request started url=%s", req.URL)

	if err := h.urlService.Process(r.Context(), req.URL); err != nil {
		logger.Printf("Error processing URL %s: %v\nStack:\n%s", req.URL, err, debug.Stack())
		response.WriteInternalError(w, err.Error())
		return
	}

	logger.Printf("Webhook request completed url=%s", req.URL)
	// Include URL in response data
	data := map[string]string{"url": req.URL}
	response.WriteSuccess(w, "URL processed successfully", data)
}
