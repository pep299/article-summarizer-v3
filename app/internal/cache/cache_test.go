package cache

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/storage"
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

// Cloud Storage Cache Tests - 実際の使用パターンに基づくカバレッジテスト

func TestCloudStorageCache(t *testing.T) {
	// 実際のバケットを使用してCloud Storageキャッシュをテスト
	testBucket := "article-summarizer-processed-articles"
	os.Setenv("CACHE_BUCKET", testBucket)
	defer os.Unsetenv("CACHE_BUCKET")

	cache, err := NewCloudStorageCache(1 * time.Hour)
	if err != nil {
		t.Skipf("Skipping Cloud Storage test: %v", err)
	}
	defer cache.Close()
	defer teardownCache(t, cache, "tmp-index-test.json")

	// Start with empty cache for test and use test-specific file
	cache.memoryIndex = make(map[string]*CacheEntry)
	cache.indexFile = "tmp-index-test.json"

	ctx := context.Background()

	// Test Set and Exists (実際のアプリケーションで使用されるパターン)
	entry := &CacheEntry{
		Title:         "Coverage Test Article",
		URL:           "http://example.com/coverage-test",
		Source:        "coverage-test-source",
		PubDate:       time.Now(),
		ProcessedDate: time.Now(),
	}

	testKey := "coverage-test-key"

	// Test Set (MarkAsProcessedで使用)
	err = cache.Set(ctx, testKey, entry)
	if err != nil {
		t.Fatalf("Failed to set cache entry: %v", err)
	}

	// Test Exists (IsCachedで使用)
	exists, err := cache.Exists(ctx, testKey)
	if err != nil {
		t.Fatalf("Failed to check existence: %v", err)
	}
	if !exists {
		t.Error("Expected key to exist")
	}

	// Test non-existent key (実際のケース)
	exists, err = cache.Exists(ctx, "non-existent-key")
	if err != nil {
		t.Fatalf("Failed to check non-existent key: %v", err)
	}
	if exists {
		t.Error("Expected non-existent key to not exist")
	}
}

func TestCloudStorageCacheManager(t *testing.T) {
	// 実際のアプリケーションでの使用パターンをテスト
	testBucket := "article-summarizer-processed-articles"
	os.Setenv("CACHE_BUCKET", testBucket)
	defer os.Unsetenv("CACHE_BUCKET")

	manager, err := NewManager("cloud-storage", 1*time.Hour)
	if err != nil {
		t.Skipf("Skipping Cloud Storage manager test: %v", err)
	}
	defer manager.Close()

	ctx := context.Background()

	item := rss.Item{
		Title:      "Coverage Test Article",
		Link:       "http://example.com/coverage-test",
		GUID:       "coverage-test-guid",
		Source:     "coverage-test-source",
		ParsedDate: time.Now(),
	}

	// Setup clean test state with empty cache
	if cache, ok := manager.cache.(*CloudStorageCache); ok {
		cache.memoryIndex = make(map[string]*CacheEntry)
		cache.indexFile = "tmp-index-test-manager.json"
		defer teardownCache(t, cache, "tmp-index-test-manager.json")
	}

	// Test IsCached (未処理の場合)
	cached, err := manager.IsCached(ctx, item)
	if err != nil {
		t.Fatalf("Failed to check if cached: %v", err)
	}
	if cached {
		t.Error("Expected item to not be cached initially")
	}

	// Test MarkAsProcessed (処理済みマーク)
	err = manager.MarkAsProcessed(ctx, item)
	if err != nil {
		t.Fatalf("Failed to mark as processed: %v", err)
	}

	// Test IsCached (処理済みの場合)
	cached, err = manager.IsCached(ctx, item)
	if err != nil {
		t.Fatalf("Failed to check if cached after processing: %v", err)
	}
	if !cached {
		t.Error("Expected item to be cached after marking as processed")
	}

	// Cleanup
	key := GenerateKey(item)
	if csCache, ok := manager.cache.(*CloudStorageCache); ok {
		err = csCache.Delete(ctx, key)
		if err != nil {
			t.Logf("Cleanup warning: Failed to delete test entry: %v", err)
		}
	}
}

func TestCloudStorageCacheLoadExistingData(t *testing.T) {
	// Test loading existing data from index-test.json (read-side test)
	testBucket := "article-summarizer-processed-articles"
	os.Setenv("CACHE_BUCKET", testBucket)
	defer os.Unsetenv("CACHE_BUCKET")

	// Create cache with empty state first
	cache := &CloudStorageCache{
		client:     nil, // Will be set properly
		bucketName: testBucket,
		indexFile:  "index-test.json",
	}

	// Initialize client
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		t.Skipf("Skipping Cloud Storage read test: %v", err)
	}
	cache.client = client
	defer cache.Close()

	// Load existing data from index-test.json
	err = cache.loadIndex(ctx)
	if err != nil {
		t.Skipf("index-test.json not found or failed to load: %v", err)
	}

	// Test: Verify known keys exist (from pre-populated index-test.json)
	knownKeys := []string{
		"test-article-1",
		"test-article-2",
		"http://tech.blog/article-123",
	}

	for _, key := range knownKeys {
		exists, err := cache.Exists(ctx, key)
		if err != nil {
			t.Fatalf("Failed to check existence of key '%s': %v", key, err)
		}
		if !exists {
			t.Errorf("Expected key '%s' to exist in index-test.json", key)
		}
	}

	// Test: Verify unknown key doesn't exist
	exists, err := cache.Exists(ctx, "unknown-key-12345")
	if err != nil {
		t.Fatalf("Failed to check non-existent key: %v", err)
	}
	if exists {
		t.Error("Expected unknown key to not exist")
	}

	// Test: Verify Get() works with pre-loaded data
	entry, err := cache.Get(ctx, "test-article-1")
	if err != nil {
		t.Fatalf("Failed to get known entry: %v", err)
	}
	if entry.Title != "Pre-existing Test Article 1" {
		t.Errorf("Expected title 'Pre-existing Test Article 1', got '%s'", entry.Title)
	}
	if entry.Source != "test-source-1" {
		t.Errorf("Expected source 'test-source-1', got '%s'", entry.Source)
	}

	// Test: Verify Get() returns cache miss for unknown key
	_, err = cache.Get(ctx, "unknown-key-12345")
	if err != ErrCacheMiss {
		t.Errorf("Expected ErrCacheMiss for unknown key, got %v", err)
	}
}

// teardownCache cleans up test files from Cloud Storage
func teardownCache(t *testing.T, cache *CloudStorageCache, filename string) {
	ctx := context.Background()
	bucket := cache.client.Bucket(cache.bucketName)

	// Delete test index file
	if err := bucket.Object(filename).Delete(ctx); err != nil && err != storage.ErrObjectNotExist {
		t.Logf("Cleanup warning: Failed to delete %s: %v", filename, err)
	}
}
