package middleware

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"os"
	"strings"
)

// AdminAuth middleware protects admin endpoints with HTTP Basic Auth
func AdminAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get admin credentials from environment
		adminUser := os.Getenv("ADMIN_USERNAME")
		adminPass := os.Getenv("ADMIN_PASSWORD")

		// If not configured, use defaults (only for development!)
		if adminUser == "" {
			adminUser = "admin"
		}
		if adminPass == "" {
			adminPass = "admin"
		}

		// Get Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			requireAuth(w)
			return
		}

		// Parse Basic Auth
		const prefix = "Basic "
		if !strings.HasPrefix(auth, prefix) {
			requireAuth(w)
			return
		}

		// Decode credentials
		payload, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
		if err != nil {
			requireAuth(w)
			return
		}

		// Split username:password
		pair := strings.SplitN(string(payload), ":", 2)
		if len(pair) != 2 {
			requireAuth(w)
			return
		}

		username := pair[0]
		password := pair[1]

		// Constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(username), []byte(adminUser)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(password), []byte(adminPass)) == 1

		if !usernameMatch || !passwordMatch {
			requireAuth(w)
			return
		}

		// Authenticated, proceed
		next.ServeHTTP(w, r)
	})
}

func requireAuth(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="EVE SDE Admin"`)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error":"authentication required"}`))
}
