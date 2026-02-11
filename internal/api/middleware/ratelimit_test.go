package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	tests := []struct {
		name      string
		key       string
		limit     int
		requests  int
		wantAllow int // how many requests should be allowed
	}{
		{
			name:      "first request creates bucket with full limit",
			key:       "key1",
			limit:     10,
			requests:  1,
			wantAllow: 1,
		},
		{
			name:      "multiple requests within limit",
			key:       "key2",
			limit:     5,
			requests:  5,
			wantAllow: 5,
		},
		{
			name:      "requests exceed limit",
			key:       "key3",
			limit:     3,
			requests:  5,
			wantAllow: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed := 0
			for i := 0; i < tt.requests; i++ {
				remaining := rl.allow(tt.key, tt.limit)
				if remaining >= 0 {
					allowed++
				}
			}

			if allowed != tt.wantAllow {
				t.Errorf("allow() allowed %d requests, want %d", allowed, tt.wantAllow)
			}
		})
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	key := "test-key"
	limit := 60 // 60 requests per minute = 1 per second

	// Exhaust all tokens
	for i := 0; i < limit; i++ {
		rl.allow(key, limit)
	}

	// Try one more - should be denied
	if remaining := rl.allow(key, limit); remaining >= 0 {
		t.Error("expected request to be denied after exhausting tokens")
	}

	// Wait for refill (60 tokens/min = 1 token/second, wait 2 seconds to get ~2 tokens)
	time.Sleep(2100 * time.Millisecond)

	// Should be allowed now (at least 2 tokens refilled)
	if remaining := rl.allow(key, limit); remaining < 0 {
		t.Error("expected request to be allowed after refill")
	}
}

func TestRateLimiter_MultipleKeys(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	limit := 5

	// Exhaust tokens for key1
	for i := 0; i < limit; i++ {
		rl.allow("key1", limit)
	}

	// key2 should still have full limit
	remaining := rl.allow("key2", limit)
	if remaining != limit-1 {
		t.Errorf("key2 should have %d remaining, got %d", limit-1, remaining)
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	rl := NewRateLimiter()
	defer rl.Stop()

	// Create some buckets
	rl.allow("key1", 10)
	rl.allow("key2", 10)

	if len(rl.requests) != 2 {
		t.Errorf("expected 2 buckets, got %d", len(rl.requests))
	}

	// Cleanup goroutine should run, but buckets won't be removed yet
	// because they were just created (cleanup removes buckets older than 2 minutes)
}
