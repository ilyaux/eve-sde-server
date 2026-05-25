package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthSkipsAdminRoutes(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{name: "admin dashboard", path: "/admin"},
		{name: "admin API", path: "/api/admin/keys"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			handler := Auth(nil, map[string]bool{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				w.WriteHeader(http.StatusNoContent)
			}))

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			if !called {
				t.Fatal("expected next handler to be called")
			}
			if w.Code != http.StatusNoContent {
				t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
			}
		})
	}
}
