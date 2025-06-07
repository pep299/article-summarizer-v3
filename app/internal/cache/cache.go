package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

// Cache interface defines cache operations
type Cache interface {
	Get(ctx context.Context, key string) (*CacheEntry, error)
	Set(ctx context.Context, key string, entry *CacheEntry) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	GetStats(ctx context.Context) (*Stats, error)
	Close() error
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Title         string    `json:"title"`
	URL           string    `json:"url"`
	Source        string    `json:"source"`
	PubDate       time.Time `json:"pub_date"`
	ProcessedDate time.Time `json:"processed_date"`
}

// Stats represents cache statistics
type Stats struct {
	TotalEntries   int           `json:"total_entries"`
	HitCount       int64         `json:"hit_count"`
	MissCount      int64         `json:"miss_count"`
	HitRate        float64       `json:"hit_rate"`
	MemoryUsage    int64         `json:"memory_usage_bytes"`
	OldestEntry    time.Time     `json:"oldest_entry"`
	AverageAge     time.Duration `json:"average_age"`
	ExpiredEntries int           `json:"expired_entries"`
}

// MemoryCache implements in-memory cache
type MemoryCache struct {
	entries     map[string]*CacheEntry
	mutex       sync.RWMutex
	duration    time.Duration
	hitCount    int64
	missCount   int64
	stopCleanup chan struct{}
	closed      bool
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(duration time.Duration) *MemoryCache {
	cache := &MemoryCache{
		entries:     make(map[string]*CacheEntry),
		duration:    duration,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// CloudStorageCache implements cache using Google Cloud Storage with JSON format
type CloudStorageCache struct {
	client      *storage.Client
	bucketName  string
	memoryIndex map[string]*CacheEntry // in-memory index cache
	indexFile   string                 // configurable for testing
}

const indexFileName = "index.json"

// NewCloudStorageCache creates a new Cloud Storage cache
func NewCloudStorageCache() (*CloudStorageCache, error) {
	ctx := context.Background()
	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating storage client: %w", err)
	}

	// Get bucket name from environment (default: article-summarizer-cache)
	bucketName := "article-summarizer-cache"
	if env := os.Getenv("CACHE_BUCKET"); env != "" {
		bucketName = env
	}

	cache := &CloudStorageCache{
		client:     client,
		bucketName: bucketName,
		indexFile:  indexFileName,
	}

	// Load index.json at startup
	if err := cache.loadIndex(ctx); err != nil {
		// If index doesn't exist or fails to load, start with empty cache
		cache.memoryIndex = make(map[string]*CacheEntry)
	}

	return cache, nil
}

// Get retrieves an entry from memory index
func (c *CloudStorageCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	if entry, exists := c.memoryIndex[key]; exists {
		return entry, nil
	}
	return nil, ErrCacheMiss
}

// Set stores an entry in memory index and saves to Cloud Storage
func (c *CloudStorageCache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	// Update memory index
	if entry.ProcessedDate.IsZero() {
		entry.ProcessedDate = time.Now()
	}
	c.memoryIndex[key] = entry

	// Save index to Cloud Storage immediately
	data, err := json.Marshal(c.memoryIndex)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	obj := c.client.Bucket(c.bucketName).Object(c.indexFile)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("writing index data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing index writer: %w", err)
	}

	return nil
}

// Delete removes an entry from memory index and updates Cloud Storage
func (c *CloudStorageCache) Delete(ctx context.Context, key string) error {
	// Remove from memory index
	delete(c.memoryIndex, key)

	// Save updated index to Cloud Storage
	data, err := json.Marshal(c.memoryIndex)
	if err != nil {
		return fmt.Errorf("marshaling index: %w", err)
	}

	obj := c.client.Bucket(c.bucketName).Object(c.indexFile)
	writer := obj.NewWriter(ctx)
	writer.ContentType = "application/json"

	if _, err := writer.Write(data); err != nil {
		writer.Close()
		return fmt.Errorf("writing index data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing index writer: %w", err)
	}

	return nil
}

// Exists checks if an entry exists in memory index (optimized)
func (c *CloudStorageCache) Exists(ctx context.Context, key string) (bool, error) {
	_, exists := c.memoryIndex[key]
	return exists, nil
}

// GetStats returns cache statistics for Cloud Storage
func (c *CloudStorageCache) GetStats(ctx context.Context) (*Stats, error) {
	stats := &Stats{
		TotalEntries: len(c.memoryIndex),
		HitCount:     0, // Not tracked in Cloud Storage implementation
		MissCount:    0, // Not tracked in Cloud Storage implementation
		HitRate:      0, // Not tracked in Cloud Storage implementation
	}

	// Calculate memory usage and ages from memory index
	var totalAge time.Duration
	now := time.Now()

	for _, entry := range c.memoryIndex {
		stats.MemoryUsage += estimateMemoryUsage(entry)

		if stats.OldestEntry.IsZero() || entry.ProcessedDate.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.ProcessedDate
		}

		totalAge += now.Sub(entry.ProcessedDate)
	}

	if stats.TotalEntries > 0 {
		stats.AverageAge = totalAge / time.Duration(stats.TotalEntries)
	}

	return stats, nil
}

// loadIndex loads the index.json from Cloud Storage into memory
func (c *CloudStorageCache) loadIndex(ctx context.Context) error {
	obj := c.client.Bucket(c.bucketName).Object(c.indexFile)

	reader, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			// Index doesn't exist yet, start with empty index
			c.memoryIndex = make(map[string]*CacheEntry)
			return nil
		}
		return fmt.Errorf("opening index reader: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("reading index data: %w", err)
	}

	if err := json.Unmarshal(data, &c.memoryIndex); err != nil {
		return fmt.Errorf("unmarshaling index: %w", err)
	}

	return nil
}

// Close closes the Cloud Storage client
func (c *CloudStorageCache) Close() error {
	return c.client.Close()
}

// Get retrieves an entry from cache
func (c *MemoryCache) Get(ctx context.Context, key string) (*CacheEntry, error) {
	// First check with read lock
	c.mutex.RLock()
	entry, exists := c.entries[key]
	if !exists {
		c.mutex.RUnlock()
		c.mutex.Lock()
		c.missCount++
		c.mutex.Unlock()
		return nil, ErrCacheMiss
	}

	c.mutex.RUnlock()

	// Need write lock to update access information
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Double-check entry still exists after re-acquiring lock
	entry, exists = c.entries[key]
	if !exists {
		c.missCount++
		return nil, ErrCacheMiss
	}

	c.hitCount++

	return entry, nil
}

// Set stores an entry in cache
func (c *MemoryCache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry.ProcessedDate.IsZero() {
		entry.ProcessedDate = time.Now()
	}

	c.entries[key] = entry
	return nil
}

// Delete removes an entry from cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.entries, key)
	return nil
}

// Exists checks if an entry exists in cache
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	_, exists := c.entries[key]
	return exists, nil
}

// Clear removes all entries from cache
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.entries = make(map[string]*CacheEntry)
	c.hitCount = 0
	c.missCount = 0
	return nil
}

// GetStats returns cache statistics
func (c *MemoryCache) GetStats(ctx context.Context) (*Stats, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	stats := &Stats{
		TotalEntries: len(c.entries),
		HitCount:     c.hitCount,
		MissCount:    c.missCount,
	}

	if c.hitCount+c.missCount > 0 {
		stats.HitRate = float64(c.hitCount) / float64(c.hitCount+c.missCount)
	}

	// Calculate memory usage estimate
	for _, entry := range c.entries {
		stats.MemoryUsage += estimateMemoryUsage(entry)
	}

	// Find oldest entry and calculate average age
	var totalAge time.Duration
	now := time.Now()

	for _, entry := range c.entries {
		if stats.OldestEntry.IsZero() || entry.ProcessedDate.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.ProcessedDate
		}

		totalAge += now.Sub(entry.ProcessedDate)
	}

	if len(c.entries) > 0 {
		stats.AverageAge = totalAge / time.Duration(len(c.entries))
	}

	stats.ExpiredEntries = 0

	return stats, nil
}

// cleanup removes expired entries periodically
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCleanup:
			return
		case <-ticker.C:
			// No cleanup needed since expiration is disabled
		}
	}
}

// Close stops the cleanup goroutine and closes the cache
func (c *MemoryCache) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.closed {
		return nil // Already closed
	}

	c.closed = true
	close(c.stopCleanup)
	return nil
}

// MarkAsProcessed marks an RSS item as processed (convenience function for CloudStorageCache)
func MarkAsProcessed(ctx context.Context, cache Cache, item rss.Item) error {
	key := GenerateKey(item)
	entry := &CacheEntry{
		Title:         item.Title,
		URL:           key, // Normalized URL
		Source:        item.Source,
		PubDate:       item.ParsedDate,
		ProcessedDate: time.Now(),
	}

	return cache.Set(ctx, key, entry)
}

// IsCached checks if an RSS item is already cached (convenience function for CloudStorageCache)
func IsCached(ctx context.Context, cache Cache, item rss.Item) (bool, error) {
	key := GenerateKey(item)
	return cache.Exists(ctx, key)
}

// normalizeURL normalizes URL by removing query parameters
func normalizeURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	// Remove query parameters
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""

	// Keep trailing slash as is
	return parsedURL.String(), nil
}

// GenerateKey generates a cache key for an RSS item
func GenerateKey(item rss.Item) string {
	// Use GUID if available, otherwise use link
	identifier := item.GUID
	if identifier == "" {
		identifier = item.Link
	}

	// Normalize URL
	normalizedURL, err := normalizeURL(identifier)
	if err != nil {
		// Fallback to original identifier if normalization fails
		return strings.TrimSpace(identifier)
	}

	return normalizedURL
}

// estimateMemoryUsage estimates memory usage of a cache entry without JSON marshaling
func estimateMemoryUsage(entry *CacheEntry) int64 {
	size := int64(len(entry.Title))
	size += int64(len(entry.URL))
	size += int64(len(entry.Source))

	// Add estimated overhead for time.Time fields and other overhead
	size += 64 // rough estimate for time.Time fields

	return size
}

// Common cache errors
var (
	ErrCacheMiss    = fmt.Errorf("cache miss")
	ErrCacheExpired = fmt.Errorf("cache entry expired")
)
