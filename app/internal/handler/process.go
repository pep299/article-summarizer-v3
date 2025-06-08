package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/middleware"
	"github.com/pep299/article-summarizer-v3/internal/service"
)

type ProcessHandler struct {
	service *service.FeedService
}

func NewProcessHandler(service *service.FeedService) *ProcessHandler {
	return &ProcessHandler{
		service: service,
	}
}

type ProcessRequest struct {
	FeedName string `json:"feedName"`
}

func (r *ProcessRequest) Validate() error {
	if r.FeedName == "" {
		return &ValidationError{Field: "feedName", Message: "FeedName is required"}
	}
	return nil
}

type ProcessContext struct {
	w http.ResponseWriter
	r *http.Request
}

func (c *ProcessContext) Request() *http.Request {
	return c.r
}

func (c *ProcessContext) JSON(code int, obj interface{}) error {
	c.w.Header().Set("Content-Type", "application/json")
	c.w.WriteHeader(code)
	return json.NewEncoder(c.w).Encode(obj)
}

func (c *ProcessContext) Bind(obj interface{}) error {
	return json.NewDecoder(c.r.Body).Decode(obj)
}

func (h *ProcessHandler) Handle(w http.ResponseWriter, r *http.Request) error {
	c := &ProcessContext{w: w, r: r}
	
	var req ProcessRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, middleware.ErrorResponse{Error: "Invalid JSON"})
	}

	if err := req.Validate(); err != nil {
		return c.JSON(400, middleware.ErrorResponse{Error: err.Error()})
	}

	if err := h.service.ProcessFeed(c.Request().Context(), req.FeedName); err != nil {
		return c.JSON(500, middleware.ErrorResponse{Error: err.Error()})
	}

	return c.JSON(200, SuccessResponse{
		Status:  "success",
		Message: "Feed processed successfully",
	})
}