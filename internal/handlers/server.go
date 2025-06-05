package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/pep299/article-summarizer-v3/internal/cache"
	"github.com/pep299/article-summarizer-v3/internal/config"
	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
	"github.com/pep299/article-summarizer-v3/internal/slack"
)

// Server holds the HTTP server and its dependencies
type Server struct {
	config       *config.Config
	rssClient    *rss.Client
	geminiClient *gemini.Client
	slackClient  *slack.Client
	cacheManager *cache.Manager
}

// NewServer creates a new HTTP server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize cache manager
	cacheManager, err := cache.NewManager(cfg.CacheType, time.Duration(cfg.CacheDuration)*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("creating cache manager: %w", err)
	}

	return &Server{
		config:       cfg,
		rssClient:    rss.NewClient(),
		geminiClient: gemini.NewClient(cfg.GeminiAPIKey, cfg.GeminiModel),
		slackClient:  slack.NewClient(cfg.SlackWebhookURL, cfg.SlackChannel),
		cacheManager: cacheManager,
	}, nil
}

// SetupRoutes configures HTTP routes
func (s *Server) SetupRoutes() *mux.Router {
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(s.corsMiddleware)
	api.Use(s.loggingMiddleware)

	// Health check
	api.HandleFunc("/health", s.healthHandler).Methods("GET")
	
	// RSS operations
	api.HandleFunc("/rss/fetch", s.fetchRSSHandler).Methods("GET")
	api.HandleFunc("/rss/process", s.processRSSHandler).Methods("POST")
	
	// Summary operations
	api.HandleFunc("/summary", s.createSummaryHandler).Methods("POST")
	api.HandleFunc("/summary/batch", s.batchSummaryHandler).Methods("POST")
	
	// Cache operations
	api.HandleFunc("/cache/stats", s.cacheStatsHandler).Methods("GET")
	api.HandleFunc("/cache/clear", s.cacheClearHandler).Methods("DELETE")
	
	// Slack operations
	api.HandleFunc("/slack/notify", s.notifySlackHandler).Methods("POST")
	
	// Webhook endpoint
	api.HandleFunc("/webhook/summarize", s.webhookSummarizeHandler).Methods("POST")
	
	// Status and configuration
	api.HandleFunc("/status", s.statusHandler).Methods("GET")
	api.HandleFunc("/config", s.configHandler).Methods("GET")

	return r
}

// healthHandler provides health check endpoint
func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"version":   "v3.0.0",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ProcessAndNotify processes RSS feeds and sends notifications to Slack
func (s *Server) ProcessAndNotify(ctx context.Context) error {
	log.Println("Starting RSS processing and notification...")
	
	// Fetch RSS feeds
	feeds, feedErrors := s.rssClient.FetchMultipleFeeds(ctx, s.config.RSSFeeds)
	
	if len(feedErrors) > 0 {
		log.Printf("RSS feed errors: %v", feedErrors)
	}
	
	// Collect all items
	var allItems []rss.Item
	for _, feed := range feeds {
		allItems = append(allItems, feed.Items...)
	}
	
	// Remove duplicates
	uniqueItems := rss.GetUniqueItems(allItems)
	log.Printf("Found %d unique articles", len(uniqueItems))
	
	// Filter out cached items
	uncachedItems, err := s.cacheManager.FilterCached(ctx, uniqueItems)
	if err != nil {
		return fmt.Errorf("filtering cached items: %w", err)
	}
	
	log.Printf("Processing %d new articles", len(uncachedItems))
	
	if len(uncachedItems) == 0 {
		log.Println("No new articles to process")
		return nil
	}
	
	// Summarize uncached items
	summaries, summaryErrors := s.geminiClient.SummarizeRSSItems(ctx, uncachedItems, s.config.MaxConcurrentRequests)
	
	// Cache new summaries and prepare Slack notifications
	var articleSummaries []slack.ArticleSummary
	successCount := 0
	
	for i, item := range uncachedItems {
		if summaryErrors[i] == nil {
			// Cache the summary
			if err := s.cacheManager.SetSummary(ctx, item, summaries[i]); err != nil {
				log.Printf("Error caching summary for %s: %v", item.Title, err)
			}
			
			// Prepare for Slack notification
			articleSummaries = append(articleSummaries, slack.ArticleSummary{
				RSS:     item,
				Summary: summaries[i],
			})
			successCount++
		} else {
			log.Printf("Error summarizing %s: %v", item.Title, summaryErrors[i])
		}
	}
	
	// Send notifications to Slack
	if len(articleSummaries) > 0 {
		if err := s.slackClient.SendMultipleSummaries(ctx, articleSummaries); err != nil {
			log.Printf("Error sending Slack notifications: %v", err)
		} else {
			log.Printf("Sent %d article summaries to Slack", len(articleSummaries))
		}
	}
	
	log.Printf("Processing complete: %d successful, %d errors", successCount, len(uncachedItems)-successCount)
	return nil
}

// Middleware functions

// corsMiddleware adds CORS headers
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware logs HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Wrap the ResponseWriter to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		
		next.ServeHTTP(wrapped, r)
		
		duration := time.Since(start)
		log.Printf("%s %s %d %v", r.Method, r.URL.Path, wrapped.statusCode, duration)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
