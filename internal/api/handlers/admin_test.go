package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ilyaux/eve-sde-server/internal/auth"
	_ "modernc.org/sqlite"
)

func TestAdminHandler_ListKeysMasksKeyValues(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			rate_limit INTEGER NOT NULL DEFAULT 60,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			active BOOLEAN NOT NULL DEFAULT 1
		);
		INSERT INTO api_keys (key, name, rate_limit, active)
		VALUES ('esk_test_key_value_that_must_not_leak', 'test key', 60, 1);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	handler := NewAdminHandler(db, auth.NewManager(db))
	req := httptest.NewRequest(http.MethodGet, "/api/admin/keys", nil)
	w := httptest.NewRecorder()

	handler.ListKeys(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "must_not_leak") {
		t.Fatalf("response leaked raw API key: %s", w.Body.String())
	}

	var response []map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response) != 1 {
		t.Fatalf("expected one key, got %d", len(response))
	}
	if response[0]["key"] == "esk_test_key_value_that_must_not_leak" {
		t.Fatal("key field contains raw API key")
	}
}
