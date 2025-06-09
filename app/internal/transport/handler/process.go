package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pep299/article-summarizer-v3/internal/infrastructure"
	"github.com/pep299/article-summarizer-v3/internal/service"
)

type Process struct {
	feedService *service.Feed
	config      *infrastructure.Config
}

func NewProcess(feedService *service.Feed, config *infrastructure.Config) *Process {
	return &Process{
		feedService: feedService,
		config:      config,
	}
}

type processRequest struct {
	FeedName string `json:"feedName"`
}

func (h *Process) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req processRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "Invalid JSON"})
		return
	}

	if req.FeedName == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: "FeedName is required"})
		return
	}

	// フィード設定を取得
	var feedURL, displayName string
	switch req.FeedName {
	case "hatena":
		feedURL = h.config.HatenaRSSURL
		displayName = "はてブ テクノロジー"
	case "lobsters":
		feedURL = h.config.LobstersRSSURL
		displayName = "Lobsters"
	default:
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errorResponse{Error: fmt.Sprintf("Unknown feed: %s", req.FeedName)})
		return
	}

	if err := h.feedService.Process(r.Context(), req.FeedName, feedURL, displayName); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errorResponse{Error: err.Error()})
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(successResponse{
		Status:  "success",
		Message: "Feed processed successfully",
	})
}