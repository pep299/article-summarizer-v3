package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/rss"
)

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()
	ctx := context.Background()

	// Test Set and Get
	entry := &CacheEntry{
		Title:         "Test Article",
		URL:           "http://example.com/test",
		Source:        "test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
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

	if retrieved.Title != entry.Title {
		t.Errorf("Expected title '%s', got '%s'", entry.Title, retrieved.Title)
	}

	if retrieved.URL != entry.URL {
		t.Errorf("Expected URL '%s', got '%s'", entry.URL, retrieved.URL)
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

func TestMemoryCacheDelete(t *testing.T) {
	cache := NewMemoryCache(1 * time.Hour)
	defer cache.Close()
	ctx := context.Background()

	entry := &CacheEntry{
		Title:         "Test Article",
		URL:           "http://example.com/test",
		Source:        "test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
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
			Title:         "Test Article",
			URL:           "http://example.com/test",
			Source:        "test-source",
			PubDate:       time.Now(),
			ProcessedDate: time.Now(),
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
		Title:         "Test Article",
		URL:           "http://example.com/test",
		Source:        "test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
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

	if stats.MissCount != 1 {
		t.Errorf("Expected 1 miss, got %d", stats.MissCount)
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
		Title:      "Test Article",
		Link:       "http://example.com/test",
		GUID:       "test-guid",
		Source:     "test-source",
		ParsedDate: time.Now(),
	}

	// Test MarkAsProcessed
	err = manager.MarkAsProcessed(ctx, item)
	if err != nil {
		t.Fatalf("Failed to mark as processed: %v", err)
	}

	// Test IsCached
	cached, err := manager.IsCached(ctx, item)
	if err != nil {
		t.Fatalf("Failed to check if cached: %v", err)
	}
	if !cached {
		t.Error("Expected item to be cached")
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
				Link:  "http://example.com/test?param=value",
				GUID:  "http://example.com/test?param=value",
			},
		},
		{
			name: "without GUID",
			item: rss.Item{
				Title: "Test Article",
				Link:  "http://example.com/test?param=value",
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

			// Key should be consistent for the same item
			key2 := GenerateKey(test.item)
			if key != key2 {
				t.Errorf("Expected consistent key generation, got '%s' and '%s'", key, key2)
			}

			// Key should be normalized URL (no query params)
			expectedKey := "http://example.com/test"
			if key != expectedKey {
				t.Errorf("Expected normalized key '%s', got '%s'", expectedKey, key)
			}
		})
	}
}

func TestEstimateMemoryUsage(t *testing.T) {
	entry := &CacheEntry{
		Title:         "Test Article Title",
		URL:           "http://example.com/test",
		Source:        "test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
	}

	size := estimateMemoryUsage(entry)

	if size <= 0 {
		t.Error("Expected positive memory usage estimate")
	}

	// Should include at least the length of strings
	minExpected := int64(len(entry.Title) + len(entry.URL) + len(entry.Source))

	if size < minExpected {
		t.Errorf("Expected memory usage to be at least %d, got %d", minExpected, size)
	}
}

// Cloud Storage Cache Tests

func TestCloudStorageCache(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Cloud Storage tests in short mode")
	}

	// Set up test bucket environment variable
	testBucket := "test-article-summarizer-cache"
	os.Setenv("CACHE_BUCKET", testBucket)
	defer os.Unsetenv("CACHE_BUCKET")

	cache, err := NewCloudStorageCache(1 * time.Hour)
	if err != nil {
		t.Skipf("Skipping Cloud Storage test: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Test Set and Get
	entry := &CacheEntry{
		Title:         "Test Article",
		URL:           "http://example.com/test",
		Source:        "test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
	}

	err = cache.Set(ctx, "test-key", entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test Get
	retrieved, err := cache.Get(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to get cache entry: %v", err)
	}

	if retrieved.Title != entry.Title {
		t.Errorf("Expected title '%s', got '%s'", entry.Title, retrieved.Title)
	}

	if retrieved.URL != entry.URL {
		t.Errorf("Expected URL '%s', got '%s'", entry.URL, retrieved.URL)
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

	// Cleanup
	err = cache.Delete(ctx, "test-key")
	if err != nil {
		t.Fatalf("Failed to delete test entry: %v", err)
	}
}

func TestCloudStorageCacheDelete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Cloud Storage tests in short mode")
	}

	testBucket := "test-article-summarizer-cache"
	os.Setenv("CACHE_BUCKET", testBucket)
	defer os.Unsetenv("CACHE_BUCKET")

	cache, err := NewCloudStorageCache(1 * time.Hour)
	if err != nil {
		t.Skipf("Skipping Cloud Storage test: %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	entry := &CacheEntry{
		Title:         "Test Article",
		URL:           "http://example.com/test",
		Source:        "test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
	}

	err = cache.Set(ctx, "test-key", entry)
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

func TestCacheManagerCloudStorage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Cloud Storage tests in short mode")
	}

	testBucket := "test-article-summarizer-cache"
	os.Setenv("CACHE_BUCKET", testBucket)
	defer os.Unsetenv("CACHE_BUCKET")

	manager, err := NewManager("cloud-storage", 1*time.Hour)
	if err != nil {
		t.Skipf("Skipping Cloud Storage test: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	item := rss.Item{
		Title:      "Test Article",
		Link:       "http://example.com/test",
		GUID:       "test-guid",
		Source:     "test-source",
		ParsedDate: time.Now(),
	}

	// Test MarkAsProcessed
	err = manager.MarkAsProcessed(ctx, item)
	if err != nil {
		t.Fatalf("Failed to mark as processed: %v", err)
	}

	// Test IsCached
	cached, err := manager.IsCached(ctx, item)
	if err != nil {
		t.Fatalf("Failed to check if cached: %v", err)
	}
	if !cached {
		t.Error("Expected item to be cached")
	}

	// Cleanup
	key := GenerateKey(item)
	if csCache, ok := manager.cache.(*CloudStorageCache); ok {
		err = csCache.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed to delete test entry: %v", err)
		}
	}
}

func TestCloudStorageCacheInvalidBucket(t *testing.T) {
	// Test with empty bucket name should use default
	os.Setenv("CACHE_BUCKET", "")
	defer os.Unsetenv("CACHE_BUCKET")

	cache, err := NewCloudStorageCache(1 * time.Hour)
	if err != nil {
		t.Skipf("Expected to create cache with default bucket name: %v", err)
	}
	if cache != nil {
		cache.Close()
	}
}
