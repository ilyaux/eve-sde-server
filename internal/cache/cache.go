package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

// Cache represents a caching layer
type Cache interface {
	Get(ctx context.Context, key string, dest interface{}) error
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	client *redis.Client
	prefix string
}

// NewRedisCache creates a new Redis cache
func NewRedisCache(addr, password string, db int, prefix string) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Info().Str("addr", addr).Msg("redis cache connected")

	return &RedisCache{
		client: client,
		prefix: prefix,
	}, nil
}

// Get retrieves a value from cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.prefix + key

	data, err := c.client.Get(ctx, fullKey).Bytes()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// Set stores a value in cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.prefix + key

	data, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, fullKey, data, ttl).Err()
}

// Delete removes a value from cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.prefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// Clear removes all cached values with the prefix
func (c *RedisCache) Clear(ctx context.Context) error {
	pattern := c.prefix + "*"

	iter := c.client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		if err := c.client.Del(ctx, iter.Val()).Err(); err != nil {
			return err
		}
	}

	return iter.Err()
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// ErrCacheMiss is returned when a key is not found in cache
var ErrCacheMiss = redis.Nil
