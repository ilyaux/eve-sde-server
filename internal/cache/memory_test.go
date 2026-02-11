package cache

import (
	"context"
	"testing"
	"time"
)

func TestNewMemoryCache(t *testing.T) {
	cache, err := NewMemoryCache(60*time.Second, 100)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	if cache == nil {
		t.Error("expected cache to be created")
	}
}

func TestMemoryCache_SetAndGet(t *testing.T) {
	cache, err := NewMemoryCache(60*time.Second, 100)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := map[string]string{"foo": "bar"}

	// Set value
	err = cache.Set(ctx, key, value, 60*time.Second)
	if err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	// Get value
	var result map[string]string
	err = cache.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if result["foo"] != "bar" {
		t.Errorf("expected value 'bar', got %s", result["foo"])
	}
}

func TestMemoryCache_GetMiss(t *testing.T) {
	cache, err := NewMemoryCache(60*time.Second, 100)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	var result string
	err = cache.Get(ctx, "non-existent", &result)

	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss, got %v", err)
	}
}

func TestMemoryCache_Delete(t *testing.T) {
	cache, err := NewMemoryCache(60*time.Second, 100)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set and verify
	cache.Set(ctx, key, value, 60*time.Second)

	var result string
	cache.Get(ctx, key, &result)
	if result != value {
		t.Errorf("expected value %s, got %s", value, result)
	}

	// Delete
	err = cache.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify deletion
	err = cache.Get(ctx, key, &result)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss after delete, got %v", err)
	}
}

func TestMemoryCache_Clear(t *testing.T) {
	cache, err := NewMemoryCache(60*time.Second, 100)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	ctx := context.Background()

	// Set multiple values
	cache.Set(ctx, "key1", "value1", 60*time.Second)
	cache.Set(ctx, "key2", "value2", 60*time.Second)

	// Clear all
	err = cache.Clear(ctx)
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	// Verify all cleared
	var result string
	err = cache.Get(ctx, "key1", &result)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss for key1 after clear, got %v", err)
	}

	err = cache.Get(ctx, "key2", &result)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss for key2 after clear, got %v", err)
	}
}

func TestMemoryCache_TTL(t *testing.T) {
	// Create cache with short TTL for testing
	cache, err := NewMemoryCache(100*time.Millisecond, 100)
	if err != nil {
		t.Fatalf("NewMemoryCache() error = %v", err)
	}
	defer cache.Close()

	ctx := context.Background()
	key := "test-key"
	value := "test-value"

	// Set value with short TTL
	cache.Set(ctx, key, value, 100*time.Millisecond)

	// Immediate get should succeed
	var result string
	err = cache.Get(ctx, key, &result)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	// Wait for expiration
	time.Sleep(200 * time.Millisecond)

	// Should be expired now
	err = cache.Get(ctx, key, &result)
	if err != ErrCacheMiss {
		t.Errorf("expected ErrCacheMiss after TTL, got %v", err)
	}
}
