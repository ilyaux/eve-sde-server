package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/allegro/bigcache/v3"
	"github.com/rs/zerolog/log"
)

// MemoryCache implements Cache using in-memory storage
type MemoryCache struct {
	cache *bigcache.BigCache
}

// NewMemoryCache creates a new in-memory cache with optimized configuration
func NewMemoryCache(ttl time.Duration, maxSizeMB int) (*MemoryCache, error) {
	config := bigcache.DefaultConfig(ttl)
	config.HardMaxCacheSize = maxSizeMB
	config.Verbose = false
	config.Shards = 1024                     // More shards = less lock contention
	config.MaxEntriesInWindow = 1000 * 10 * 60  // Optimize for high throughput
	config.MaxEntrySize = 500                // Limit individual entry size (500 bytes)
	config.CleanWindow = 1 * time.Minute     // Clean expired entries every minute

	cache, err := bigcache.New(context.Background(), config)
	if err != nil {
		return nil, err
	}

	log.Info().
		Dur("ttl", ttl).
		Int("max_size_mb", maxSizeMB).
		Int("shards", config.Shards).
		Msg("in-memory cache created with optimized configuration")

	return &MemoryCache{cache: cache}, nil
}

// Get retrieves a value from cache
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.cache.Get(key)
	if err == bigcache.ErrEntryNotFound {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set stores a value in cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.cache.Set(key, data)
}

// Delete removes a value from cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	return c.cache.Delete(key)
}

// Clear removes all cached values
func (c *MemoryCache) Clear(ctx context.Context) error {
	return c.cache.Reset()
}

// Close closes the cache
func (c *MemoryCache) Close() error {
	return c.cache.Close()
}

// Stats returns cache statistics
func (c *MemoryCache) Stats() bigcache.Stats {
	return c.cache.Stats()
}
