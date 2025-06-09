package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/service"
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

type errorResponse struct {
	Error string `json:"error"`
}

type successResponse struct {
	Status  string `json:"status"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message"`
}

func (h *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req webhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "Invalid JSON"})
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "URL is required"})
		return
	}

	if err := h.urlService.Process(r.Context(), req.URL); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(successResponse{
		Status:  "success",
		URL:     req.URL,
		Message: "URL processed successfully",
	})
}
