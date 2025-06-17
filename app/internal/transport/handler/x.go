package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"

	"github.com/pep299/article-summarizer-v3/internal/repository"
	"github.com/pep299/article-summarizer-v3/internal/transport/response"
)

type X struct {
	client repository.Client
}

type XQuoteChain struct {
	client repository.Client
}

func NewX(s repository.Client) *X {
	return &X{
		client: s,
	}
}

func NewXQuoteChain(s repository.Client) *XQuoteChain {
	return &XQuoteChain{
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

func (h *XQuoteChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logger := log.New(funcframework.LogWriter(r.Context()), "", 0)

	// Get URL parameter
	urlParam := r.URL.Query().Get("url")
	if urlParam == "" {
		logger.Printf("Missing URL parameter in X quote chain request")
		response.WriteBadRequest(w, "URL parameter is required")
		return
	}

	// Check if URL is supported
	if !h.client.IsSupported(urlParam) {
		logger.Printf("Unsupported URL format for quote chain: %s", urlParam)
		response.WriteBadRequest(w, "Unsupported URL format")
		return
	}

	logger.Printf("X quote chain request started url=%s", urlParam)

	// Get the quote chain
	quoteChain, err := h.client.FetchQuoteChain(r.Context(), urlParam)
	if err != nil {
		logger.Printf("Error fetching X quote chain %s: %v", urlParam, err)
		response.WriteInternalError(w, err.Error())
		return
	}

	logger.Printf("X quote chain request completed url=%s chain_length=%d", urlParam, len(quoteChain))

	// Build text response
	var textResponse strings.Builder
	for i, post := range quoteChain {
		textResponse.WriteString(post.URL)
		textResponse.WriteString("\n")
		textResponse.WriteString(post.Text)
		textResponse.WriteString("\n")

		// Add blank line between posts (except for the last one)
		if i < len(quoteChain)-1 {
			textResponse.WriteString("\n")
		}
	}

	// Write plain text response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(textResponse.String()))
}
