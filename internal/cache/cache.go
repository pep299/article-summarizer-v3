package cache

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pep299/article-summarizer-v3/internal/gemini"
	"github.com/pep299/article-summarizer-v3/internal/rss"
)

// Cache interface defines cache operations
type Cache interface {
	Get(ctx context.Context, key string) (*CacheEntry, error)
	Set(ctx context.Context, key string, entry *CacheEntry) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
	Clear(ctx context.Context) error
	GetStats(ctx context.Context) (*Stats, error)
	Close() error
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key        string                     `json:"key"`
	RSS        rss.Item                   `json:"rss"`
	Summary    gemini.SummarizeResponse   `json:"summary"`
	CreatedAt  time.Time                  `json:"created_at"`
	ExpiresAt  time.Time                  `json:"expires_at"`
	AccessedAt time.Time                  `json:"accessed_at"`
	AccessCount int                       `json:"access_count"`
}

// Stats represents cache statistics
type Stats struct {
	TotalEntries    int           `json:"total_entries"`
	HitCount        int64         `json:"hit_count"`
	MissCount       int64         `json:"miss_count"`
	HitRate         float64       `json:"hit_rate"`
	MemoryUsage     int64         `json:"memory_usage_bytes"`
	OldestEntry     time.Time     `json:"oldest_entry"`
	AverageAge      time.Duration `json:"average_age"`
	ExpiredEntries  int           `json:"expired_entries"`
}

// MemoryCache implements in-memory cache
type MemoryCache struct {
	entries     map[string]*CacheEntry
	mutex       sync.RWMutex
	duration    time.Duration
	hitCount    int64
	missCount   int64
	stopCleanup chan struct{}
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
	
	// Check if expired
	now := time.Now()
	if now.After(entry.ExpiresAt) {
		c.mutex.RUnlock()
		// Need write lock to delete expired entry
		c.mutex.Lock()
		// Double-check after acquiring write lock
		if entry, exists := c.entries[key]; exists && now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
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
	if !exists || now.After(entry.ExpiresAt) {
		c.missCount++
		return nil, ErrCacheMiss
	}
	
	// Update access information
	entry.AccessedAt = now
	entry.AccessCount++
	c.hitCount++
	
	return entry, nil
}

// Set stores an entry in cache
func (c *MemoryCache) Set(ctx context.Context, key string, entry *CacheEntry) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	entry.Key = key
	entry.CreatedAt = now
	entry.ExpiresAt = now.Add(c.duration)
	entry.AccessedAt = now
	entry.AccessCount = 0
	
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
	
	entry, exists := c.entries[key]
	if !exists {
		return false, nil
	}
	
	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		return false, nil
	}
	
	return true, nil
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
	var expiredCount int
	now := time.Now()
	
	for _, entry := range c.entries {
		if stats.OldestEntry.IsZero() || entry.CreatedAt.Before(stats.OldestEntry) {
			stats.OldestEntry = entry.CreatedAt
		}
		
		totalAge += now.Sub(entry.CreatedAt)
		
		if now.After(entry.ExpiresAt) {
			expiredCount++
		}
	}
	
	if len(c.entries) > 0 {
		stats.AverageAge = totalAge / time.Duration(len(c.entries))
	}
	
	stats.ExpiredEntries = expiredCount
	
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
			c.cleanupExpired()
		}
	}
}

// cleanupExpired removes expired entries
func (c *MemoryCache) cleanupExpired() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	now := time.Now()
	for key, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, key)
		}
	}
}

// Close stops the cleanup goroutine and closes the cache
func (c *MemoryCache) Close() error {
	close(c.stopCleanup)
	return nil
}

// Manager handles cache operations with convenience methods
type Manager struct {
	cache Cache
}

// NewManager creates a new cache manager
func NewManager(cacheType string, duration time.Duration) (*Manager, error) {
	var cache Cache
	
	switch cacheType {
	case "memory":
		cache = NewMemoryCache(duration)
	default:
		return nil, fmt.Errorf("unsupported cache type: %s", cacheType)
	}
	
	return &Manager{cache: cache}, nil
}

// GetSummary retrieves a cached summary for an RSS item
func (m *Manager) GetSummary(ctx context.Context, item rss.Item) (*gemini.SummarizeResponse, error) {
	key := GenerateKey(item)
	entry, err := m.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	
	return &entry.Summary, nil
}

// SetSummary caches a summary for an RSS item
func (m *Manager) SetSummary(ctx context.Context, item rss.Item, summary gemini.SummarizeResponse) error {
	key := GenerateKey(item)
	entry := &CacheEntry{
		RSS:     item,
		Summary: summary,
	}
	
	return m.cache.Set(ctx, key, entry)
}

// IsCached checks if an RSS item is already cached
func (m *Manager) IsCached(ctx context.Context, item rss.Item) (bool, error) {
	key := GenerateKey(item)
	return m.cache.Exists(ctx, key)
}

// FilterCached filters out already cached items from a list
func (m *Manager) FilterCached(ctx context.Context, items []rss.Item) ([]rss.Item, error) {
	var uncached []rss.Item
	
	for _, item := range items {
		cached, err := m.IsCached(ctx, item)
		if err != nil {
			return nil, err
		}
		
		if !cached {
			uncached = append(uncached, item)
		}
	}
	
	return uncached, nil
}

// GetCachedSummaries retrieves cached summaries for a list of items
func (m *Manager) GetCachedSummaries(ctx context.Context, items []rss.Item) ([]CacheEntry, error) {
	var summaries []CacheEntry
	
	for _, item := range items {
		key := GenerateKey(item)
		entry, err := m.cache.Get(ctx, key)
		if err != nil {
			continue // Skip if not cached
		}
		
		summaries = append(summaries, *entry)
	}
	
	return summaries, nil
}

// GetStats returns cache statistics
func (m *Manager) GetStats(ctx context.Context) (*Stats, error) {
	return m.cache.GetStats(ctx)
}

// Clear clears all cached entries
func (m *Manager) Clear(ctx context.Context) error {
	return m.cache.Clear(ctx)
}

// Close closes the cache and stops background goroutines
func (m *Manager) Close() error {
	return m.cache.Close()
}

// GenerateKey generates a cache key for an RSS item
func GenerateKey(item rss.Item) string {
	// Use GUID if available, otherwise use link
	identifier := item.GUID
	if identifier == "" {
		identifier = item.Link
	}
	
	// Create MD5 hash for consistent key length
	hash := md5.Sum([]byte(identifier))
	return fmt.Sprintf("article:%x", hash)
}

// estimateMemoryUsage estimates memory usage of a cache entry without JSON marshaling
func estimateMemoryUsage(entry *CacheEntry) int64 {
	size := int64(len(entry.Key))
	size += int64(len(entry.RSS.Title) + len(entry.RSS.Link) + len(entry.RSS.Description) + len(entry.RSS.GUID))
	size += int64(len(entry.Summary.Summary) + len(entry.Summary.KeyPoints))
	
	// Add estimated overhead for struct fields and slices
	size += 128 // rough estimate for time.Time fields and other overhead
	
	// Add memory for categories and key points
	for _, category := range entry.RSS.Category {
		size += int64(len(category))
	}
	
	return size
}

// Common cache errors
var (
	ErrCacheMiss = fmt.Errorf("cache miss")
	ErrCacheExpired = fmt.Errorf("cache entry expired")
)
