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
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	// Read body for detailed error logging
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Printf("Error reading request body: %v", err)
		response.WriteInternalError(w, "Error reading request")
		return
	}

	var req processRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		logger.Printf("Invalid JSON in process request: %v, body: %s", err, string(bodyBytes))
		response.WriteBadRequest(w, "Invalid JSON")
		return
	}

	if req.FeedName == "" {
		logger.Printf("Missing feedName in process request")
		response.WriteBadRequest(w, "FeedName is required")
		return
	}

	logger.Printf("Process request started feed=%s", req.FeedName)

	if err := h.feedService.Process(r.Context(), req.FeedName); err != nil {
		logger.Printf("Error processing feed %s: %v", req.FeedName, err)
		response.WriteInternalError(w, err.Error())
		return
	}

	logger.Printf("Process request completed feed=%s", req.FeedName)
	response.WriteSuccess(w, "Feed processed successfully", nil)
}
