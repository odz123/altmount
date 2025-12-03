package cache

import (
	"sync"
	"time"

	metapb "github.com/javi11/altmount/internal/metadata/proto"
)

// MetadataCacheEntry holds a cached metadata entry with expiration
type MetadataCacheEntry struct {
	Metadata  *metapb.FileMetadata
	ExpiresAt time.Time
}

// MetadataCache provides an LRU-style cache for file metadata
// to reduce disk I/O for frequently accessed files
type MetadataCache struct {
	mu        sync.RWMutex
	cache     map[string]*MetadataCacheEntry
	ttl       time.Duration
	maxSize   int
	hits      int64
	misses    int64
	evictions int64
}

// NewMetadataCache creates a new metadata cache
func NewMetadataCache(ttl time.Duration, maxSize int) *MetadataCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute // Default 5 minute TTL
	}
	if maxSize <= 0 {
		maxSize = 10000 // Default 10k entries
	}

	cache := &MetadataCache{
		cache:   make(map[string]*MetadataCacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// Start background cleanup
	go cache.cleanupLoop()

	return cache
}

// Get retrieves metadata from cache, returns nil if not found or expired
func (c *MetadataCache) Get(path string) *metapb.FileMetadata {
	c.mu.RLock()
	entry, exists := c.cache[path]
	c.mu.RUnlock()

	if !exists {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil
	}

	if time.Now().After(entry.ExpiresAt) {
		// Expired, remove and return nil
		c.mu.Lock()
		delete(c.cache, path)
		c.misses++
		c.mu.Unlock()
		return nil
	}

	c.mu.Lock()
	c.hits++
	c.mu.Unlock()
	return entry.Metadata
}

// Set stores metadata in cache
func (c *MetadataCache) Set(path string, metadata *metapb.FileMetadata) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict oldest entries if at capacity
	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[path] = &MetadataCacheEntry{
		Metadata:  metadata,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes a specific path from cache
func (c *MetadataCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, path)
}

// InvalidatePrefix removes all entries matching a path prefix
func (c *MetadataCache) InvalidatePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for path := range c.cache {
		if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
			delete(c.cache, path)
		}
	}
}

// Clear removes all entries from cache
func (c *MetadataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*MetadataCacheEntry)
}

// Stats returns cache statistics
func (c *MetadataCache) Stats() (hits, misses, evictions int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.evictions, len(c.cache)
}

// evictOldest removes the oldest 10% of entries (called with lock held)
func (c *MetadataCache) evictOldest() {
	var oldestPaths []string
	var oldestTime time.Time

	// Find expired entries first
	now := time.Now()
	for path, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			oldestPaths = append(oldestPaths, path)
		}
	}

	// If not enough expired, find oldest by expiry time
	if len(oldestPaths) < c.maxSize/10 {
		// Simple eviction: remove 10% of entries
		toEvict := c.maxSize / 10
		if toEvict < 1 {
			toEvict = 1
		}

		for path, entry := range c.cache {
			if oldestTime.IsZero() || entry.ExpiresAt.Before(oldestTime) {
				oldestTime = entry.ExpiresAt
			}
			if len(oldestPaths) < toEvict {
				oldestPaths = append(oldestPaths, path)
			}
		}
	}

	for _, path := range oldestPaths {
		delete(c.cache, path)
		c.evictions++
	}
}

// cleanupLoop periodically removes expired entries
func (c *MetadataCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		c.cleanup()
	}
}

// cleanup removes expired entries
func (c *MetadataCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for path, entry := range c.cache {
		if now.After(entry.ExpiresAt) {
			delete(c.cache, path)
		}
	}
}

// DirectoryCacheEntry holds cached directory listing
type DirectoryCacheEntry struct {
	Files     []string
	Dirs      []string
	ExpiresAt time.Time
}

// DirectoryCache provides caching for directory listings
type DirectoryCache struct {
	mu      sync.RWMutex
	cache   map[string]*DirectoryCacheEntry
	ttl     time.Duration
	maxSize int
}

// NewDirectoryCache creates a new directory cache
func NewDirectoryCache(ttl time.Duration, maxSize int) *DirectoryCache {
	if ttl <= 0 {
		ttl = 30 * time.Second // Default 30 second TTL for directory listings
	}
	if maxSize <= 0 {
		maxSize = 1000 // Default 1k directories
	}

	cache := &DirectoryCache{
		cache:   make(map[string]*DirectoryCacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}

	go cache.cleanupLoop()

	return cache
}

// Get retrieves directory listing from cache
func (c *DirectoryCache) Get(path string) (files, dirs []string, found bool) {
	c.mu.RLock()
	entry, exists := c.cache[path]
	c.mu.RUnlock()

	if !exists || time.Now().After(entry.ExpiresAt) {
		return nil, nil, false
	}

	return entry.Files, entry.Dirs, true
}

// Set stores directory listing in cache
func (c *DirectoryCache) Set(path string, files, dirs []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		// Simple eviction: clear half the cache
		count := 0
		for p := range c.cache {
			delete(c.cache, p)
			count++
			if count >= c.maxSize/2 {
				break
			}
		}
	}

	c.cache[path] = &DirectoryCacheEntry{
		Files:     files,
		Dirs:      dirs,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

// Invalidate removes a path and all its children from cache
func (c *DirectoryCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, path)

	// Also invalidate children
	for p := range c.cache {
		if len(p) > len(path) && p[:len(path)] == path {
			delete(c.cache, p)
		}
	}
}

// cleanupLoop periodically removes expired entries
func (c *DirectoryCache) cleanupLoop() {
	ticker := time.NewTicker(c.ttl)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for path, entry := range c.cache {
			if now.After(entry.ExpiresAt) {
				delete(c.cache, path)
			}
		}
		c.mu.Unlock()
	}
}
