package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/ilya/eve-sde-server/internal/auth"
	"github.com/rs/zerolog/log"
)

type contextKey string

const APIKeyContextKey contextKey = "api_key"

// Auth middleware validates API keys
func Auth(authManager *auth.Manager, public map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for public endpoints
			if public[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for /api/esi/* (ESI proxy)
			if strings.HasPrefix(r.URL.Path, "/api/esi/") {
				next.ServeHTTP(w, r)
				return
			}

			// Skip auth for /api/admin/* (protected by AdminAuth middleware instead)
			if strings.HasPrefix(r.URL.Path, "/api/admin/") {
				next.ServeHTTP(w, r)
				return
			}

			// Extract API key from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Support both "Bearer <key>" and raw key
			apiKey := strings.TrimPrefix(authHeader, "Bearer ")
			apiKey = strings.TrimSpace(apiKey)

			if apiKey == "" {
				http.Error(w, `{"error":"invalid authorization header"}`, http.StatusUnauthorized)
				return
			}

			// Validate API key
			key, err := authManager.ValidateAPIKey(r.Context(), apiKey)
			if err != nil {
				log.Warn().
					Str("key_hash", hashAPIKey(apiKey)).
					Err(err).
					Msg("Invalid API key")
				http.Error(w, `{"error":"invalid or expired API key"}`, http.StatusUnauthorized)
				return
			}

			// Add API key to context for rate limiting
			ctx := context.WithValue(r.Context(), APIKeyContextKey, key)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetAPIKey retrieves the API key from request context
func GetAPIKey(ctx context.Context) (*auth.APIKey, bool) {
	key, ok := ctx.Value(APIKeyContextKey).(*auth.APIKey)
	return key, ok
}

// hashAPIKey creates a SHA256 hash of API key for safe logging
func hashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])[:16] // First 16 chars of hash
}
