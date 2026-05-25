package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDiffHandler_GetChangelog(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE items (
			type_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			volume REAL
		);
		CREATE TABLE sde_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version TEXT NOT NULL,
			checksum TEXT NOT NULL UNIQUE,
			downloaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			imported_at TIMESTAMP,
			import_duration_seconds INTEGER,
			items_count INTEGER,
			error TEXT
		);
		INSERT INTO sde_versions (version, checksum, imported_at, items_count)
		VALUES ('20260525', 'abc123', '2026-05-25T10:00:00Z', 42000);
	`)
	if err != nil {
		t.Fatalf("create schema: %v", err)
	}

	handler := NewDiffHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/changelog", nil)
	w := httptest.NewRecorder()

	handler.GetChangelog(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Versions []struct {
			Version    string `json:"version"`
			ImportedAt string `json:"imported_at"`
			ItemCount  int    `json:"item_count"`
		} `json:"versions"`
		Count int `json:"count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Count != 1 || len(response.Versions) != 1 {
		t.Fatalf("expected one version, got count=%d len=%d", response.Count, len(response.Versions))
	}
	if response.Versions[0].Version != "20260525" || response.Versions[0].ItemCount != 42000 {
		t.Fatalf("unexpected version response: %+v", response.Versions[0])
	}
}
