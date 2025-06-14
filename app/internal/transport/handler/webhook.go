package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

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

	// Read body for detailed error logging
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Printf("Error reading request body: %v", err)
		response.WriteInternalError(w, "Error reading request")
		return
	}

	var req webhookRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		logger.Printf("Invalid JSON in webhook request: %v, body: %s", err, string(bodyBytes))
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
		logger.Printf("Error processing URL %s: %v", req.URL, err)
		response.WriteInternalError(w, err.Error())
		return
	}

	logger.Printf("Webhook request completed url=%s", req.URL)
	// Include URL in response data
	data := map[string]string{"url": req.URL}
	response.WriteSuccess(w, "URL processed successfully", data)
}
