package service

import (
	"context"
	"testing"
	"time"

	"github.com/huylvt/gisty/internal/repository"
)

func setupTestCache(t *testing.T) (*Cache, func()) {
	ctx := context.Background()

	redisClient, err := repository.NewRedisClient(ctx, "redis://localhost:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}

	cache := NewCache(redisClient)

	cleanup := func() {
		// Clean up test keys
		redisClient.Client.Del(ctx, "paste:test001", "paste:test002", "paste:test003", "paste:test004", "paste:test005")
		redisClient.Close()
	}

	return cache, cleanup
}

func TestCache_SetAndGet(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test001"
	content := "Hello, World!"

	// Set content
	err := cache.Set(ctx, shortID, content, time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get content
	retrieved, found, err := cache.Get(ctx, shortID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Error("Get() should find the key")
	}
	if retrieved != content {
		t.Errorf("Get() = %q, want %q", retrieved, content)
	}
}

func TestCache_Get_NotFound(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()

	_, found, err := cache.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found {
		t.Error("Get() should not find nonexistent key")
	}
}

func TestCache_Delete(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test002"

	// Set content
	err := cache.Set(ctx, shortID, "test content", time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it exists
	exists, err := cache.Exists(ctx, shortID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Key should exist after Set")
	}

	// Delete
	err = cache.Delete(ctx, shortID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify it's gone
	exists, err = cache.Exists(ctx, shortID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Key should not exist after Delete")
	}
}

func TestCache_Exists(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test003"

	// Check non-existent
	exists, err := cache.Exists(ctx, shortID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Error("Key should not exist before Set")
	}

	// Set content
	err = cache.Set(ctx, shortID, "test content", time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Check exists
	exists, err = cache.Exists(ctx, shortID)
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if !exists {
		t.Error("Key should exist after Set")
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test004"

	// Set with very short TTL
	err := cache.Set(ctx, shortID, "test content", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Verify it exists
	_, found, err := cache.Get(ctx, shortID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !found {
		t.Error("Key should exist immediately after Set")
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	// Verify it's expired
	_, found, err = cache.Get(ctx, shortID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found {
		t.Error("Key should not exist after TTL expiration")
	}
}

func TestCache_GetTTL(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test005"
	ttl := 5 * time.Minute

	// Set with TTL
	err := cache.Set(ctx, shortID, "test content", ttl)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get TTL
	remainingTTL, err := cache.GetTTL(ctx, shortID)
	if err != nil {
		t.Fatalf("GetTTL() error = %v", err)
	}

	// TTL should be close to what we set (allowing for some time to pass)
	if remainingTTL < 4*time.Minute || remainingTTL > ttl {
		t.Errorf("GetTTL() = %v, want close to %v", remainingTTL, ttl)
	}
}

func TestCache_Refresh(t *testing.T) {
	cache, cleanup := setupTestCache(t)
	defer cleanup()

	ctx := context.Background()
	shortID := "test005"

	// Set with short TTL
	err := cache.Set(ctx, shortID, "test content", time.Minute)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Refresh with longer TTL
	newTTL := 10 * time.Minute
	err = cache.Refresh(ctx, shortID, newTTL)
	if err != nil {
		t.Fatalf("Refresh() error = %v", err)
	}

	// Verify new TTL
	remainingTTL, err := cache.GetTTL(ctx, shortID)
	if err != nil {
		t.Fatalf("GetTTL() error = %v", err)
	}

	if remainingTTL < 9*time.Minute || remainingTTL > newTTL {
		t.Errorf("After Refresh(), TTL = %v, want close to %v", remainingTTL, newTTL)
	}
}

func TestCache_SetWithDefaultTTL(t *testing.T) {
	ctx := context.Background()

	redisClient, err := repository.NewRedisClient(ctx, "redis://localhost:6379")
	if err != nil {
		t.Skipf("Redis not available: %v", err)
	}
	defer redisClient.Close()

	customTTL := 30 * time.Minute
	cache := NewCacheWithTTL(redisClient, customTTL)

	shortID := "test_default_ttl"

	// Set with default TTL
	err = cache.SetWithDefaultTTL(ctx, shortID, "test content")
	if err != nil {
		t.Fatalf("SetWithDefaultTTL() error = %v", err)
	}
	defer cache.Delete(ctx, shortID)

	// Verify TTL is close to custom default
	remainingTTL, err := cache.GetTTL(ctx, shortID)
	if err != nil {
		t.Fatalf("GetTTL() error = %v", err)
	}

	if remainingTTL < 29*time.Minute || remainingTTL > customTTL {
		t.Errorf("Default TTL = %v, want close to %v", remainingTTL, customTTL)
	}
}