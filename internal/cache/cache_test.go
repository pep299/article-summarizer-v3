package cache

import (
	"context"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()
	ctx := context.Background()

	// Test Set and Get
	entry := &CacheEntry{
		RSS: rss.Item{
			Title: "Test Article",
			Link:  "http://example.com/test",
		},
		Summary: gemini.SummarizeResponse{
			Summary:   "Test summary",
			KeyPoints: "Test key points",
		},
	}

	err := cache.Set(ctx, "test-key", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test Get
	retrieved, err := cache.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to get cache entry: %v", err)
	}

	if retrieved.RSS.Title != entry.RSS.Title {
		t.Errorf("Expected title '%s', got '%s'", entry.RSS.Title, retrieved.RSS.Title)
	}

	if retrieved.Summary.Summary != entry.Summary.Summary {
		t.Errorf("Expected summary '%s', got '%s'", entry.Summary.Summary, retrieved.Summary.Summary)
	}

	// Test Exists
	exists, err := cache.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	// Test non-existent key
	exists, err = cache.Exists(ctx, "non-existent")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist")
	}

	// Test Get non-existent key
	_, err = cache.Get(ctx, "non-existent")
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss, got %v", err)
	}
}

func TestMemoryCacheExpiration(t *testing.T) {
	cache := NewMemoryCache(50 * time.Millisecond)
	defer cache.Close()
	ctx := context.Background()

	entry := &CacheEntry{
		RSS: rss.Item{
			Title: "Test Article",
			Link:  "http://example.com/test",
		},
		Summary: gemini.SummarizeResponse{
			Summary: "Test summary",
		},
	}

	err := cache.Set(ctx, "test-key", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Should exist immediately
	exists, err := cache.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist immediately after setting")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not exist after expiration
	exists, err = cache.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after expiration")
	}

	// Get should return cache miss
	_, err = cache.Get(ctx, "test-key")
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss after expiration, got %v", err)
	}
}

func TestMemoryCacheDelete(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()
	ctx := context.Background()

	entry := &CacheEntry{
		RSS: rss.Item{
			Title: "Test Article",
			Link:  "http://example.com/test",
		},
		Summary: gemini.SummarizeResponse{
			Summary: "Test summary",
		},
	}

	err := cache.Set(ctx, "test-key", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Delete the entry
	err = cache.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to delete cache entry: %v", err)
	}

	// Should not exist after deletion
	exists, err := cache.Exists(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if exists {
		t.Error("Expected key to not exist after deletion")
	}
}

func TestMemoryCacheClear(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()
	ctx := context.Background()

	// Add multiple entries
	for i := 0; i < 3; i++ {
		entry := &CacheEntry{
			RSS: rss.Item{
				Title: "Test Article",
				Link:  "http://example.com/test",
			},
			Summary: gemini.SummarizeResponse{
				Summary: "Test summary",
			},
		}
		err := cache.Set(ctx, fmt.Sprintf("test-key-%d", i), entry)
		if err != nil {
			t.Fatalf("Failed to set cache entry %d: %v", i, err)
		}
	}

	// Clear all entries
	err := cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Failed to clear cache: %v", err)
	}

	// Check that all entries are gone
	for i := 0; i < 3; i++ {
		exists, err := cache.Exists(ctx, fmt.Sprintf("test-key-%d", i))
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if exists {
			t.Errorf("Expected key %d to not exist after clear", i)
		}
	}
}

func TestMemoryCacheStats(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()
	ctx := context.Background()

	// Add an entry
	entry := &CacheEntry{
		RSS: rss.Item{
			Title: "Test Article",
			Link:  "http://example.com/test",
		},
		Summary: gemini.SummarizeResponse{
			Summary: "Test summary",
		},
	}

	err := cache.Set(ctx, "test-key", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Get stats
	stats, err := cache.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 total entry, got %d", stats.TotalEntries)
	}

	// Trigger a hit
	_, err = cache.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to get cache entry: %v", err)
	}

	// Trigger a miss
	_, err = cache.Get(ctx, "non-existent")
	if err != ErrCacheMiss {
		t.Errorf("Expected cache miss, got %v", err)
	}

	// Get updated stats
	stats, err = cache.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get updated stats: %v", err)
	}

	if stats.HitCount != 1 {
		t.Errorf("Expected 1 hit, got %d", stats.HitCount)
	}

	if stats.MissCount != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.MissCount)
	}

	if stats.HitRate != 0.5 {
		t.Errorf("Expected hit rate 0.5, got %f", stats.HitRate)
	}
}

func TestCacheManager(t *testing.T) {
	manager, err := NewManager("memory", 1*time.Hour)
	if err != nil {
		t.Fatalf("Failed to create cache manager: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	item := rss.Item{
		Title: "Test Article",
		Link:  "http://example.com/test",
		GUID:  "test-guid",
	}

	summary := gemini.SummarizeResponse{
		Summary:   "Test summary",
		KeyPoints: "Test key points",
	}

	// Test SetSummary and GetSummary
	err = manager.SetSummary(ctx, item, summary)
	if err != nil {
		t.Fatalf("Failed to set summary: %v", err)
	}

	retrievedSummary, err := manager.GetSummary(ctx, item)
	if err != nil {
		t.Fatalf("Failed to get summary: %v", err)
	}

	if retrievedSummary.Summary != summary.Summary {
		t.Errorf("Expected summary '%s', got '%s'", summary.Summary, retrievedSummary.Summary)
	}

	// Test IsCached
	cached, err := manager.IsCached(ctx, item)
	if err != nil {
		t.Fatalf("Failed to check if cached: %v", err)
	}
	if !cached {
		t.Error("Expected item to be cached")
	}

	// Test FilterCached
	items := []rss.Item{
		item, // This should be filtered out as it's cached
		{
			Title: "New Article",
			Link:  "http://example.com/new",
			GUID:  "new-guid",
		},
	}

	uncached, err := manager.FilterCached(ctx, items)
	if err != nil {
		t.Fatalf("Failed to filter cached items: %v", err)
	}

	if len(uncached) != 1 {
		t.Errorf("Expected 1 uncached item, got %d", len(uncached))
	}

	if uncached[0].Title != "New Article" {
		t.Errorf("Expected 'New Article', got '%s'", uncached[0].Title)
	}
}

func TestGenerateKey(t *testing.T) {
	tests := []struct {
		name string
		item rss.Item
	}{
		{
			name: "with GUID",
			item: rss.Item{
				Title: "Test Article",
				Link:  "http://example.com/test",
				GUID:  "test-guid",
			},
		},
		{
			name: "without GUID",
			item: rss.Item{
				Title: "Test Article",
				Link:  "http://example.com/test",
				GUID:  "",
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := GenerateKey(test.item)
			
			if key == "" {
				t.Error("Expected non-empty key")
			}
			
			if !strings.HasPrefix(key, "article:") {
				t.Errorf("Expected key to start with 'article:', got '%s'", key)
			}
			
			// Key should be consistent for the same item
			key2 := GenerateKey(test.item)
			if key != key2 {
				t.Errorf("Expected consistent key generation, got '%s' and '%s'", key, key2)
			}
		})
	}
}

func TestEstimateMemoryUsage(t *testing.T) {
	entry := &CacheEntry{
		Key: "test-key",
		RSS: rss.Item{
			Title:       "Test Article Title",
			Link:        "http://example.com/test",
			Description: "Test description",
			GUID:        "test-guid",
			Category:    []string{"tech", "news"},
		},
		Summary: gemini.SummarizeResponse{
			Summary:   "Test summary content",
			KeyPoints: "Test key points",
		},
	}

	size := estimateMemoryUsage(entry)
	
	if size <= 0 {
		t.Error("Expected positive memory usage estimate")
	}
	
	// Should include at least the length of strings
	minExpected := int64(len(entry.Key) + len(entry.RSS.Title) + len(entry.RSS.Link) + 
		len(entry.RSS.Description) + len(entry.RSS.GUID) + len(entry.Summary.Summary) + 
		len(entry.Summary.KeyPoints))
	
	for _, category := range entry.RSS.Category {
		minExpected += int64(len(category))
	}
	
	if size < minExpected {
		t.Errorf("Expected memory usage to be at least %d, got %d", minExpected, size)
	}
}
