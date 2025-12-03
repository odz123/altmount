package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"sync"
	"time"

	"github.com/javi11/altmount/internal/database"
)

// APIKeyCache provides an in-memory cache for API key authentication
// to avoid database queries on every stream request
type APIKeyCache struct {
	userRepo    *database.UserRepository
	mu          sync.RWMutex
	hashedKeys  map[string]struct{} // Set of valid hashed API keys
	lastRefresh time.Time
	refreshTTL  time.Duration
}

// NewAPIKeyCache creates a new API key cache
func NewAPIKeyCache(userRepo *database.UserRepository, refreshTTL time.Duration) *APIKeyCache {
	if refreshTTL <= 0 {
		refreshTTL = 30 * time.Second // Default 30 second TTL
	}

	cache := &APIKeyCache{
		userRepo:   userRepo,
		hashedKeys: make(map[string]struct{}),
		refreshTTL: refreshTTL,
	}

	return cache
}

// Start begins the background refresh goroutine
func (c *APIKeyCache) Start(ctx context.Context) {
	// Initial load
	if err := c.refresh(ctx); err != nil {
		slog.ErrorContext(ctx, "Failed initial API key cache load", "error", err)
	}

	// Background refresh
	go c.backgroundRefresh(ctx)
}

// backgroundRefresh periodically refreshes the cache
func (c *APIKeyCache) backgroundRefresh(ctx context.Context) {
	ticker := time.NewTicker(c.refreshTTL)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.refresh(ctx); err != nil {
				slog.ErrorContext(ctx, "Failed to refresh API key cache", "error", err)
			}
		}
	}
}

// refresh reloads all API keys from the database
func (c *APIKeyCache) refresh(ctx context.Context) error {
	users, err := c.userRepo.GetAllUsers(ctx)
	if err != nil {
		return err
	}

	newHashedKeys := make(map[string]struct{}, len(users))
	for _, user := range users {
		if user.APIKey == nil || *user.APIKey == "" {
			continue
		}
		hashedKey := HashAPIKey(*user.APIKey)
		newHashedKeys[hashedKey] = struct{}{}
	}

	c.mu.Lock()
	c.hashedKeys = newHashedKeys
	c.lastRefresh = time.Now()
	c.mu.Unlock()

	slog.Debug("API key cache refreshed", "key_count", len(newHashedKeys))
	return nil
}

// IsValidKey checks if a hashed API key is valid (O(1) lookup)
func (c *APIKeyCache) IsValidKey(hashedKey string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, exists := c.hashedKeys[hashedKey]
	return exists
}

// Invalidate forces a cache refresh
func (c *APIKeyCache) Invalidate(ctx context.Context) error {
	return c.refresh(ctx)
}

// GetLastRefresh returns the last refresh time
func (c *APIKeyCache) GetLastRefresh() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastRefresh
}

// HashAPIKey generates a SHA256 hash of the API key
func HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}
