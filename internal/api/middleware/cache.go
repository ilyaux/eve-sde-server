package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ilya/eve-sde-server/internal/cache"
	"github.com/rs/zerolog/log"
)

// CacheMiddleware wraps handlers with caching
type CacheMiddleware struct {
	cache cache.Cache
	ttl   time.Duration
}

// NewCacheMiddleware creates a new cache middleware
func NewCacheMiddleware(c cache.Cache, ttl time.Duration) *CacheMiddleware {
	return &CacheMiddleware{
		cache: c,
		ttl:   ttl,
	}
}

// CacheResponse caches GET request responses
func (cm *CacheMiddleware) CacheResponse(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET requests
		if r.Method != http.MethodGet {
			next.ServeHTTP(w, r)
			return
		}

		// Skip health checks and metrics
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		// Generate cache key
		cacheKey := fmt.Sprintf("http:%s:%s", r.Method, r.URL.String())

		// Try to get from cache
		var cachedData []byte
		if err := cm.cache.Get(r.Context(), cacheKey, &cachedData); err == nil {
			// Cache hit
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write(cachedData)
			return
		}

		// Cache miss - capture response
		w.Header().Set("X-Cache", "MISS")
		rec := &responseRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
			body:           []byte{},
		}

		next.ServeHTTP(rec, r)

		// Cache successful responses
		if rec.statusCode == http.StatusOK && len(rec.body) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			if err := cm.cache.Set(ctx, cacheKey, rec.body, cm.ttl); err != nil {
				log.Error().Err(err).Str("key", cacheKey).Msg("failed to cache response")
			}
		}
	})
}

// responseRecorder captures the response for caching
type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       []byte
}

func (rec *responseRecorder) WriteHeader(code int) {
	rec.statusCode = code
	rec.ResponseWriter.WriteHeader(code)
}

func (rec *responseRecorder) Write(b []byte) (int, error) {
	rec.body = append(rec.body, b...)
	return rec.ResponseWriter.Write(b)
}
