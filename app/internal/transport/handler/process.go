package handler

import (
	"encoding/json"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/service"
	"github.com/pep299/article-summarizer-v3/internal/transport/response"
)

type Process struct {
	feedService *service.Feed
}

func NewProcess(feedService *service.Feed) *Process {
	return &Process{
		feedService: feedService,
	}
}

type processRequest struct {
	FeedName string `json:"feedName"`
}

func (h *Process) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req processRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.FeedName == "" {
		response.WriteBadRequest(w, "FeedName is required")
		return
	}

	if err := h.feedService.Process(r.Context(), req.FeedName); err != nil {
		response.WriteInternalError(w, err.Error())
		return
	}

	response.WriteSuccess(w, "Feed processed successfully", nil)
}
