package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

type rateLimiter struct {
	requests map[string]*bucket
	mu       sync.RWMutex
	cleanup  *time.Ticker
	stopCh   chan struct{} // Channel for graceful shutdown
}

type bucket struct {
	tokens    int
	lastRefil time.Time
	limit     int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter() *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string]*bucket),
		cleanup:  time.NewTicker(time.Minute),
		stopCh:   make(chan struct{}),
	}

	// Cleanup old entries with graceful shutdown
	go func() {
		for {
			select {
			case <-rl.cleanup.C:
				rl.mu.Lock()
				now := time.Now()
				for key, b := range rl.requests {
					if now.Sub(b.lastRefil) > 2*time.Minute {
						delete(rl.requests, key)
					}
				}
				rl.mu.Unlock()
			case <-rl.stopCh:
				log.Debug().Msg("rate limiter cleanup stopped")
				return
			}
		}
	}()

	return rl
}

// Stop gracefully stops the rate limiter cleanup goroutine
func (rl *rateLimiter) Stop() {
	close(rl.stopCh)
	rl.cleanup.Stop()
}

// RateLimit middleware enforces rate limits per API key
func RateLimit(limiter *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get API key from context
			apiKey, ok := GetAPIKey(r.Context())
			if !ok {
				// No API key in context, skip rate limiting (public endpoint)
				next.ServeHTTP(w, r)
				return
			}

			// Check rate limit
			remaining := limiter.allow(apiKey.Key, apiKey.RateLimit)
			if remaining < 0 {
				log.Warn().
					Str("key_name", apiKey.Name).
					Int("rate_limit", apiKey.RateLimit).
					Msg("Rate limit exceeded")
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				http.Error(w, `{"error":"rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			// Add rate limit headers for successful requests
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", apiKey.RateLimit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

			next.ServeHTTP(w, r)
		})
	}
}

// allow checks if request is allowed and returns remaining tokens (-1 if denied)
func (rl *rateLimiter) allow(key string, limit int) int {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	b, exists := rl.requests[key]

	if !exists {
		// Create new bucket with full capacity
		rl.requests[key] = &bucket{
			tokens:    limit,
			lastRefil: now,
			limit:     limit,
		}
		// Consume one token for current request
		rl.requests[key].tokens--
		return limit - 1
	}

	// Refill tokens based on time passed (smooth refill, per-second basis)
	elapsed := now.Sub(b.lastRefil)
	// Calculate tokens to add: (limit tokens per minute) / 60 seconds * elapsed seconds
	tokensToAdd := int(elapsed.Seconds() * float64(limit) / 60.0)

	if tokensToAdd > 0 {
		b.tokens = min(limit, b.tokens+tokensToAdd)
		b.lastRefil = now
	}

	// Check if we have tokens available
	if b.tokens > 0 {
		b.tokens--
		return b.tokens
	}

	return -1
}
