package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/transport/response"
)

type X struct {
	client repository.Client
}

func NewX(s repository.Client) *X {
	return &X{
		client: s,
	}
}

func (h *X) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	// Get URL parameter
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		logger.Printf("Missing URL parameter in X request")
		response.WriteBadRequest(w, "URL parameter is required")
		return
	}

	// Check if URL is supported
	if !h.client.IsSupported(urlParam) {
		logger.Printf("Unsupported URL format: %s", urlParam)
		response.WriteBadRequest(w, "Unsupported URL format")
		return
	}

	logger.Printf("X request started url=%s", urlParam)

	// Get the post data
	postData, err := h.client.FetchPost(r.Context(), urlParam)
	if err != nil {
		logger.Printf("Error fetching X post %s: %v", urlParam, err)
		response.WriteInternalError(w, err.Error())
		return
	}

	logger.Printf("X request completed url=%s author=%s", urlParam, postData.AuthorName)

	// Write JSON response directly
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(postData)
}
