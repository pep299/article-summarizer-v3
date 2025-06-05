package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

// fetchRSSHandler fetches RSS feeds
func (s *Server) fetchRSSHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	feeds, errors := s.rssClient.FetchMultipleFeeds(ctx, s.config.RSSFeeds)
	
	response := map[string]interface{}{
		"feeds":  feeds,
		"errors": errors,
		"count":  len(feeds),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// processRSSHandler processes RSS feeds and creates summaries
func (s *Server) processRSSHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Fetch RSS feeds
	feeds, feedErrors := s.rssClient.FetchMultipleFeeds(ctx, s.config.RSSFeeds)
	
	var allItems []rss.Item
	for _, feed := range feeds {
		allItems = append(allItems, feed.Items...)
	}
	
	// Remove duplicates
	uniqueItems := rss.GetUniqueItems(allItems)
	
	// Apply filters
	filterOptions := rss.FilterOptions{
		ExcludeCategories: []string{"ask"},  // Exclude Lobsters "ask" category
		MaxAge:            24 * time.Hour,   // Only articles from last 24 hours
		MinTitleLength:    10,               // Minimum title length
	}
	filteredItems := rss.FilterItems(uniqueItems, filterOptions)
	
	// Filter cached items
	uncachedItems, err := s.cacheManager.FilterCached(ctx, filteredItems)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error filtering cached items: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Summarize uncached items
	summaries, summaryErrors := s.geminiClient.SummarizeRSSItems(ctx, uncachedItems, s.config.MaxConcurrentRequests)
	
	// Cache new summaries
	for i, item := range uncachedItems {
		if summaryErrors[i] == nil {
			s.cacheManager.SetSummary(ctx, item, summaries[i])
		}
	}
	
	// Get all cached summaries (including newly created ones)
	cachedSummaries, _ := s.cacheManager.GetCachedSummaries(ctx, filteredItems)
	
	response := map[string]interface{}{
		"processed_items":   len(uncachedItems),
		"total_items":       len(uniqueItems),
		"filtered_items":    len(filteredItems),
		"cached_summaries":  len(cachedSummaries),
		"feed_errors":       feedErrors,
		"summary_errors":    summaryErrors,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// createSummaryHandler creates a summary for a single article
func (s *Server) createSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var req gemini.SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	summary, err := s.geminiClient.SummarizeArticle(ctx, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating summary: %v", err), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// batchSummaryHandler creates summaries for multiple articles
func (s *Server) batchSummaryHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var requests []gemini.SummarizeRequest
	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	summaries, errors := s.geminiClient.SummarizeMultipleArticles(ctx, requests, s.config.MaxConcurrentRequests)
	
	response := map[string]interface{}{
		"summaries": summaries,
		"errors":    errors,
		"count":     len(summaries),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

// webhookSummarizeHandler handles webhook requests for summarization
func (s *Server) webhookSummarizeHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	var req struct {
		URL         string `json:"url"`
		Title       string `json:"title"`
		Description string `json:"description"`
		SendToSlack bool   `json:"send_to_slack"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	
	// Create summarization request
	summarizeReq := gemini.SummarizeRequest{
		Title:       req.Title,
		Link:        req.URL,
		Description: req.Description,
	}
	
	// Generate summary
	summary, err := s.geminiClient.SummarizeArticle(ctx, summarizeReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating summary: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Send to Slack if requested
	if req.SendToSlack {
		rssItem := rss.Item{
			Title:       req.Title,
			Link:        req.URL,
			Description: req.Description,
		}
		
		articleSummary := slack.ArticleSummary{
			RSS:     rssItem,
			Summary: *summary,
		}
		
		if err := s.slackClient.SendArticleSummary(ctx, articleSummary); err != nil {
			log.Printf("Error sending to Slack: %v", err)
		}
	}
	
	response := map[string]interface{}{
		"summary":      summary,
		"sent_to_slack": req.SendToSlack,
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
		"status":     "running",
		"version":    "v3.0.0",
		"timestamp":  r.Context().Value("timestamp"),
		"cache":      cacheStats,
		"rss_feeds":  len(s.config.RSSFeeds),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// configHandler returns configuration (sanitized)
func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	// Return sanitized configuration without sensitive data
	response := map[string]interface{}{
		"port":                    s.config.Port,
		"host":                    s.config.Host,
		"gemini_model":            s.config.GeminiModel,
		"slack_channel":           s.config.SlackChannel,
		"rss_feeds":               s.config.RSSFeeds,
		"update_interval_minutes": s.config.UpdateInterval,
		"cache_type":              s.config.CacheType,
		"cache_duration_hours":    s.config.CacheDuration,
		"max_concurrent_requests": s.config.MaxConcurrentRequests,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
