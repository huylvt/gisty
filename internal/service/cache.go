package service

import (
	"context"
	"time"

	"github.com/huylvt/gisty/internal/repository"
	"github.com/redis/go-redis/v9"
)

const (
	// DefaultCacheTTL is the default TTL for cached content
	DefaultCacheTTL = 1 * time.Hour
	// CacheKeyPrefix is the prefix for all cache keys
	CacheKeyPrefix = "paste:"
)

// Cache handles caching operations using Redis
type Cache struct {
	client     *redis.Client
	defaultTTL time.Duration
}

// NewCache creates a new Cache service
func NewCache(redisClient *repository.Redis) *Cache {
	return &Cache{
		client:     redisClient.Client,
		defaultTTL: DefaultCacheTTL,
	}
}

// NewCacheWithTTL creates a new Cache service with custom default TTL
func NewCacheWithTTL(redisClient *repository.Redis, defaultTTL time.Duration) *Cache {
	return &Cache{
		client:     redisClient.Client,
		defaultTTL: defaultTTL,
	}
}

// Set stores content in cache with the specified TTL
func (c *Cache) Set(ctx context.Context, shortID, content string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	key := c.buildKey(shortID)
	return c.client.Set(ctx, key, content, ttl).Err()
}

// Get retrieves content from cache
// Returns the content, a boolean indicating if the key was found, and an error
func (c *Cache) Get(ctx context.Context, shortID string) (string, bool, error) {
	key := c.buildKey(shortID)

	content, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", false, nil // Key not found
		}
		return "", false, err
	}

	return content, true, nil
}

// Delete removes content from cache
func (c *Cache) Delete(ctx context.Context, shortID string) error {
	key := c.buildKey(shortID)
	return c.client.Del(ctx, key).Err()
}

// Exists checks if a key exists in cache
func (c *Cache) Exists(ctx context.Context, shortID string) (bool, error) {
	key := c.buildKey(shortID)

	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// SetWithDefaultTTL stores content in cache with the default TTL
func (c *Cache) SetWithDefaultTTL(ctx context.Context, shortID, content string) error {
	return c.Set(ctx, shortID, content, c.defaultTTL)
}

// GetTTL returns the remaining TTL for a key
func (c *Cache) GetTTL(ctx context.Context, shortID string) (time.Duration, error) {
	key := c.buildKey(shortID)
	return c.client.TTL(ctx, key).Result()
}

// Refresh updates the TTL of an existing key without changing its value
func (c *Cache) Refresh(ctx context.Context, shortID string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = c.defaultTTL
	}

	key := c.buildKey(shortID)
	return c.client.Expire(ctx, key, ttl).Err()
}

// buildKey constructs the cache key for a given shortID
func (c *Cache) buildKey(shortID string) string {
	return CacheKeyPrefix + shortID
}