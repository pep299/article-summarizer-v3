package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/middleware"
	"github.com/pep299/article-summarizer-v3/internal/service"
)

type WebhookHandler struct {
	service *service.URLService
}

func NewWebhookHandler(service *service.URLService) *WebhookHandler {
	return &WebhookHandler{
		service: service,
	}
}

type WebhookRequest struct {
	URL string `json:"url"`
}

func (r *WebhookRequest) Validate() error {
	if r.URL == "" {
		return &ValidationError{Field: "url", Message: "URL is required"}
	}
	return nil
}

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type SuccessResponse struct {
	Status  string `json:"status"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message"`
}

type WebhookContext struct {
	w http.ResponseWriter
	r *http.Request
}

func (c *WebhookContext) Request() *http.Request {
	return c.r
}

func (c *WebhookContext) JSON(code int, obj interface{}) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(code)
	return json.NewEncoder(c.w).Encode(obj)
}

func (c *WebhookContext) Bind(obj interface{}) error {
	return json.NewDecoder(c.r.Body).Decode(obj)
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	c := &WebhookContext{w: w, r: r}
	
	var req WebhookRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, middleware.ErrorResponse{Error: "Invalid JSON"})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(400, middleware.ErrorResponse{Error: err.Error()})
	}

	if err := h.service.ProcessURL(c.Request().Context(), req.URL); err != nil {
		return c.JSON(500, middleware.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(200, SuccessResponse{
		Status:  "success",
		URL:     req.URL,
		Message: "URL processed successfully",
	})
}