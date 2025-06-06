package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

// fetchRSSHandler fetches RSS feeds for a specific feed
func (s *Server) fetchRSSHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	feedName := vars["feed"]

	feedConfig, exists := s.config.RSSFeeds[feedName]
	if !exists {
		http.Error(w, fmt.Sprintf("Feed %s not found", feedName), http.StatusNotFound)
		return
	}

	articles, err := s.rssClient.FetchFeed(ctx, feedConfig.Name, feedConfig.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching feed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"feed":     feedConfig.Name,
		"articles": articles,
		"count":    len(articles),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// processRSSHandler processes RSS feeds and creates summaries for a specific feed
func (s *Server) processRSSHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	feedName := vars["feed"]

	err := s.ProcessSingleFeed(ctx, feedName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing feed: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"status":  "success",
		"feed":    feedName,
		"message": "Feed processed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createSummaryHandler creates a summary for a single article
func (s *Server) createSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	summary, err := s.geminiClient.SummarizeURL(ctx, req.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// cacheStatsHandler returns cache statistics
func (s *Server) cacheStatsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := s.cacheManager.GetStats(ctx)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting cache stats: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// cacheClearHandler clears the cache
func (s *Server) cacheClearHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := s.cacheManager.Clear(ctx); err != nil {
		http.Error(w, fmt.Sprintf("Error clearing cache: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"status":  "success",
		"message": "Cache cleared successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// notifySlackHandler sends a notification to Slack
func (s *Server) notifySlackHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.slackClient.SendSimpleMessage(ctx, req.Message); err != nil {
		http.Error(w, fmt.Sprintf("Error sending Slack message: %v", err), http.StatusInternalServerError)
		return
	}

	response := map[string]string{
		"status":  "success",
		"message": "Slack notification sent",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// webhookSummarizeHandler handles webhook requests for summarization (v1 style)
func (s *Server) webhookSummarizeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req struct {
		URL     string `json:"url"`
		Channel string `json:"channel,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL parameter is required", http.StatusBadRequest)
		return
	}

	// Generate summary using on-demand prompt
	summary, err := s.geminiClient.SummarizeURLForOnDemand(ctx, req.URL)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating summary: %v", err), http.StatusInternalServerError)
		return
	}

	// Create article structure for Slack notification
	article := rss.Item{
		Title:  "", // Title will be extracted by Gemini or left empty
		Link:   req.URL,
		Source: "ondemand",
	}

	// Determine target channel
	targetChannel := req.Channel
	if targetChannel == "" {
		targetChannel = s.config.WebhookSlackChannel
	}

	// Send to Slack
	if err := s.slackClient.SendOnDemandSummary(ctx, article, *summary, targetChannel); err != nil {
		log.Printf("Error sending to Slack: %v", err)
	}

	response := map[string]interface{}{
		"status":        "success",
		"summary":       summary,
		"sent_to_slack": true,
		"channel":       targetChannel,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// statusHandler returns system status
func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get cache stats
	cacheStats, _ := s.cacheManager.GetStats(ctx)

	response := map[string]interface{}{
		"status":    "running",
		"version":   "v3.0.0",
		"cache":     cacheStats,
		"rss_feeds": len(s.config.RSSFeeds),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// configHandler returns configuration (sanitized)
func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	// Return sanitized configuration without sensitive data
	response := map[string]interface{}{
		"port":                  s.config.Port,
		"host":                  s.config.Host,
		"gemini_model":          s.config.GeminiModel,
		"slack_channel":         s.config.SlackChannel,
		"webhook_slack_channel": s.config.WebhookSlackChannel,
		"rss_feeds":             s.config.RSSFeeds,
		"cache_type":            s.config.CacheType,
		"cache_duration_hours":  s.config.CacheDuration,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
