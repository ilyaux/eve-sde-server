package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestSystemHealth(t *testing.T) {
	db := newSystemTestDB(t)
	handler := NewSystemHandler(db, testVersionInfo(), time.Now().Add(-5*time.Second))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler.Health(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["status"] != "OK" {
		t.Fatalf("expected OK health status, got %v", body["status"])
	}
	if _, ok := body["uptime_seconds"].(float64); !ok {
		t.Fatal("expected uptime_seconds in health response")
	}
}

func TestSystemReady(t *testing.T) {
	db := newSystemTestDB(t)
	mustExec(t, db, `
		CREATE TABLE items (type_id INTEGER PRIMARY KEY, name TEXT NOT NULL);
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
		INSERT INTO items (type_id, name) VALUES (34, 'Tritanium');
		INSERT INTO sde_versions (version, checksum, imported_at, items_count)
		VALUES ('20260101', 'abc123', '2026-01-01T00:00:00Z', 1);
	`)

	handler := NewSystemHandler(db, testVersionInfo(), time.Now())
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var body struct {
		Status           string            `json:"status"`
		Checks           map[string]string `json:"checks"`
		ItemsCount       int               `json:"items_count"`
		LatestSDEVersion string            `json:"latest_sde_version"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "ready" {
		t.Fatalf("expected ready status, got %s", body.Status)
	}
	if body.Checks["database"] != "ok" || body.Checks["items"] != "ok" || body.Checks["sde_version"] != "ok" {
		t.Fatalf("unexpected checks: %#v", body.Checks)
	}
	if body.ItemsCount != 1 {
		t.Fatalf("expected one item, got %d", body.ItemsCount)
	}
	if body.LatestSDEVersion != "20260101" {
		t.Fatalf("expected latest SDE version, got %q", body.LatestSDEVersion)
	}
}

func TestSystemReadyReportsMissingSchema(t *testing.T) {
	db := newSystemTestDB(t)
	handler := NewSystemHandler(db, testVersionInfo(), time.Now())

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	w := httptest.NewRecorder()

	handler.Ready(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d: %s", http.StatusServiceUnavailable, w.Code, w.Body.String())
	}

	var body struct {
		Status string            `json:"status"`
		Checks map[string]string `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Status != "not_ready" {
		t.Fatalf("expected not_ready status, got %s", body.Status)
	}
	if body.Checks["items"] != "unavailable" {
		t.Fatalf("expected missing items schema to be reported, got %#v", body.Checks)
	}
}

func TestSystemVersion(t *testing.T) {
	db := newSystemTestDB(t)
	handler := NewSystemHandler(db, testVersionInfo(), time.Now())

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	handler.Version(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var body VersionInfo
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Version != "test" || body.Commit != "abc123" || body.GoVersion == "" {
		t.Fatalf("unexpected version response: %#v", body)
	}
}

func newSystemTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	return db
}

func mustExec(t *testing.T, db *sql.DB, query string) {
	t.Helper()

	if _, err := db.Exec(query); err != nil {
		t.Fatalf("exec query: %v", err)
	}
}

func testVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   "test",
		Commit:    "abc123",
		BuildDate: "2026-01-01T00:00:00Z",
		GoVersion: "go-test",
	}
}
