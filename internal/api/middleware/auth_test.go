package middleware

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ilyaux/eve-sde-server/internal/auth"
	_ "modernc.org/sqlite"
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

func TestAuthRequiresKeyForESICacheClear(t *testing.T) {
	db := setupAuthMiddlewareTestDB(t)
	defer db.Close()

	called := false
	handler := Auth(auth.NewManager(db), map[string]bool{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/esi/cache/clear", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if called {
		t.Fatal("expected cache clear request without API key not to reach next handler")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthRequiresKeyForNonGetESIProxyRequests(t *testing.T) {
	db := setupAuthMiddlewareTestDB(t)
	defer db.Close()

	called := false
	handler := Auth(auth.NewManager(db), map[string]bool{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/esi/universe/types/34", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if called {
		t.Fatal("expected non-GET ESI request without API key not to reach next handler")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthSkipsReadOnlyESIProxy(t *testing.T) {
	called := false
	handler := Auth(nil, map[string]bool{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/esi/types/34", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Fatal("expected read-only ESI request to reach next handler")
	}
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, w.Code)
	}
}

func setupAuthMiddlewareTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			rate_limit INTEGER NOT NULL DEFAULT 60,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			active BOOLEAN NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("create api_keys table: %v", err)
	}

	return db
}
