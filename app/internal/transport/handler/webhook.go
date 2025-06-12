package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"runtime/debug"

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
	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.URL == "" {
		response.WriteBadRequest(w, "URL is required")
		return
	}

	if err := h.urlService.Process(r.Context(), req.URL); err != nil {
		log.Printf("Error processing URL %s: %v\nStack:\n%s", req.URL, err, debug.Stack())
		response.WriteInternalError(w, err.Error())
		return
	}

	// Include URL in response data
	data := map[string]string{"url": req.URL}
	response.WriteSuccess(w, "URL processed successfully", data)
}
