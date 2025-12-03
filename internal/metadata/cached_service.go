package metadata

import (
	"context"
	"sync"
	"time"

	"github.com/javi11/altmount/internal/cache"
	metapb "github.com/javi11/altmount/internal/metadata/proto"
)

// CachedMetadataService wraps MetadataService with caching for improved performance
type CachedMetadataService struct {
	*MetadataService
	metadataCache  *cache.MetadataCache
	directoryCache *cache.DirectoryCache
	singleFlight   *cache.SingleFlight
}

// NewCachedMetadataService creates a new cached metadata service
func NewCachedMetadataService(service *MetadataService, metadataTTL time.Duration, dirTTL time.Duration, maxMetadataEntries, maxDirEntries int) *CachedMetadataService {
	return &CachedMetadataService{
		MetadataService: service,
		metadataCache:   cache.NewMetadataCache(metadataTTL, maxMetadataEntries),
		directoryCache:  cache.NewDirectoryCache(dirTTL, maxDirEntries),
		singleFlight:    cache.NewSingleFlight(),
	}
}

// ReadFileMetadata reads file metadata with caching and request coalescing
func (cs *CachedMetadataService) ReadFileMetadata(virtualPath string) (*metapb.FileMetadata, error) {
	// Check cache first
	if cached := cs.metadataCache.Get(virtualPath); cached != nil {
		return cached, nil
	}

	// Use singleflight to coalesce concurrent requests for the same path
	result, err, _ := cs.singleFlight.Do("meta:"+virtualPath, func() (interface{}, error) {
		// Read from disk
		metadata, err := cs.MetadataService.ReadFileMetadata(virtualPath)
		if err != nil {
			return nil, err
		}

		// Cache the result
		if metadata != nil {
			cs.metadataCache.Set(virtualPath, metadata)
		}

		return metadata, nil
	})

	if err != nil {
		return nil, err
	}

	if result == nil {
		return nil, nil
	}

	return result.(*metapb.FileMetadata), nil
}

// WriteFileMetadata writes file metadata and invalidates cache
func (cs *CachedMetadataService) WriteFileMetadata(virtualPath string, metadata *metapb.FileMetadata) error {
	if err := cs.MetadataService.WriteFileMetadata(virtualPath, metadata); err != nil {
		return err
	}

	// Invalidate and update cache
	cs.metadataCache.Set(virtualPath, metadata)

	// Invalidate parent directory cache
	parentDir := getParentDir(virtualPath)
	cs.directoryCache.Invalidate(parentDir)

	return nil
}

// UpdateFileMetadata updates metadata and invalidates cache
func (cs *CachedMetadataService) UpdateFileMetadata(virtualPath string, updateFunc func(*metapb.FileMetadata)) error {
	if err := cs.MetadataService.UpdateFileMetadata(virtualPath, updateFunc); err != nil {
		return err
	}

	// Invalidate cache (next read will refresh)
	cs.metadataCache.Invalidate(virtualPath)

	return nil
}

// DeleteFileMetadata deletes metadata and invalidates cache
func (cs *CachedMetadataService) DeleteFileMetadata(virtualPath string) error {
	if err := cs.MetadataService.DeleteFileMetadata(virtualPath); err != nil {
		return err
	}

	// Invalidate cache
	cs.metadataCache.Invalidate(virtualPath)

	// Invalidate parent directory cache
	parentDir := getParentDir(virtualPath)
	cs.directoryCache.Invalidate(parentDir)

	return nil
}

// DeleteFileMetadataWithSourceNzb deletes metadata and invalidates cache
func (cs *CachedMetadataService) DeleteFileMetadataWithSourceNzb(ctx context.Context, virtualPath string, deleteSourceNzb bool) error {
	if err := cs.MetadataService.DeleteFileMetadataWithSourceNzb(ctx, virtualPath, deleteSourceNzb); err != nil {
		return err
	}

	// Invalidate cache
	cs.metadataCache.Invalidate(virtualPath)

	// Invalidate parent directory cache
	parentDir := getParentDir(virtualPath)
	cs.directoryCache.Invalidate(parentDir)

	return nil
}

// DeleteDirectory deletes a directory and invalidates cache
func (cs *CachedMetadataService) DeleteDirectory(virtualPath string) error {
	if err := cs.MetadataService.DeleteDirectory(virtualPath); err != nil {
		return err
	}

	// Invalidate all cache entries under this path
	cs.metadataCache.InvalidatePrefix(virtualPath)
	cs.directoryCache.Invalidate(virtualPath)

	return nil
}

// ListDirectory lists directory contents with caching and request coalescing
func (cs *CachedMetadataService) ListDirectory(virtualPath string) ([]string, error) {
	// Check cache first
	if files, _, found := cs.directoryCache.Get(virtualPath); found {
		return files, nil
	}

	// Use singleflight to coalesce concurrent requests
	result, err, _ := cs.singleFlight.Do("dir:"+virtualPath, func() (interface{}, error) {
		files, err := cs.MetadataService.ListDirectory(virtualPath)
		if err != nil {
			return nil, err
		}

		// Also get subdirectories for complete cache
		dirs, _ := cs.MetadataService.ListSubdirectories(virtualPath)

		// Cache the result
		cs.directoryCache.Set(virtualPath, files, dirs)

		return files, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// ListSubdirectories lists subdirectories with caching
func (cs *CachedMetadataService) ListSubdirectories(virtualPath string) ([]string, error) {
	// Check cache first
	if _, dirs, found := cs.directoryCache.Get(virtualPath); found {
		return dirs, nil
	}

	// Use singleflight to coalesce concurrent requests
	result, err, _ := cs.singleFlight.Do("subdir:"+virtualPath, func() (interface{}, error) {
		dirs, err := cs.MetadataService.ListSubdirectories(virtualPath)
		if err != nil {
			return nil, err
		}

		// Also get files for complete cache
		files, _ := cs.MetadataService.ListDirectory(virtualPath)

		// Cache the result
		cs.directoryCache.Set(virtualPath, files, dirs)

		return dirs, nil
	})

	if err != nil {
		return nil, err
	}

	return result.([]string), nil
}

// InvalidateCache invalidates all cached data for a path
func (cs *CachedMetadataService) InvalidateCache(virtualPath string) {
	cs.metadataCache.Invalidate(virtualPath)
	cs.directoryCache.Invalidate(virtualPath)
}

// InvalidateCachePrefix invalidates all cached data under a path prefix
func (cs *CachedMetadataService) InvalidateCachePrefix(prefix string) {
	cs.metadataCache.InvalidatePrefix(prefix)
	cs.directoryCache.Invalidate(prefix)
}

// ClearCache clears all cached data
func (cs *CachedMetadataService) ClearCache() {
	cs.metadataCache.Clear()
}

// CacheStats returns cache statistics
func (cs *CachedMetadataService) CacheStats() (metaHits, metaMisses, metaEvictions int64, metaSize int) {
	return cs.metadataCache.Stats()
}

// getParentDir extracts the parent directory from a path
func getParentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			if i == 0 {
				return "/"
			}
			return path[:i]
		}
	}
	return "/"
}

// FileExistsCache provides a cached check for file existence
type FileExistsCache struct {
	mu    sync.RWMutex
	cache map[string]bool
	ttl   time.Duration
	times map[string]time.Time
}

// NewFileExistsCache creates a new file existence cache
func NewFileExistsCache(ttl time.Duration) *FileExistsCache {
	return &FileExistsCache{
		cache: make(map[string]bool),
		times: make(map[string]time.Time),
		ttl:   ttl,
	}
}

// Get checks if a file existence result is cached
func (c *FileExistsCache) Get(path string) (exists bool, found bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ts, ok := c.times[path]
	if !ok || time.Since(ts) > c.ttl {
		return false, false
	}

	return c.cache[path], true
}

// Set caches a file existence result
func (c *FileExistsCache) Set(path string, exists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[path] = exists
	c.times[path] = time.Now()
}

// Invalidate removes a path from cache
func (c *FileExistsCache) Invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, path)
	delete(c.times, path)
}
