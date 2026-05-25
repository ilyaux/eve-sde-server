package evesde

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"OK","uptime_seconds":12,"timestamp":"2026-01-01T00:00:00Z"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	status, err := client.HealthStatus()
	if err != nil {
		t.Fatalf("HealthStatus() error = %v", err)
	}
	if status.Status != "OK" || status.UptimeSeconds != 12 {
		t.Fatalf("unexpected health status: %#v", status)
	}

	healthy, err := client.Health()
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}
	if !healthy {
		t.Fatal("expected Health() to return true")
	}
}

func TestReady(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ready" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status":"ready",
			"checks":{"database":"ok","items":"ok","sde_version":"not_imported"},
			"items_count":8,
			"latest_sde_version":null,
			"latest_sde_imported_at":null,
			"version":{"version":"dev","commit":"abc123","build_date":"unknown","go_version":"go1.25"}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	ready, err := client.Ready()
	if err != nil {
		t.Fatalf("Ready() error = %v", err)
	}
	if ready.Status != "ready" || ready.ItemsCount != 8 {
		t.Fatalf("unexpected readiness response: %#v", ready)
	}
	if ready.Checks["database"] != "ok" {
		t.Fatalf("unexpected readiness checks: %#v", ready.Checks)
	}
	if ready.LatestSDEVersion != nil || ready.LatestSDEImportedAt != nil {
		t.Fatalf("expected nil latest SDE values: %#v", ready)
	}
}

func TestVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/version" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"version":"1.2.3","commit":"abc123","build_date":"2026-01-01T00:00:00Z","go_version":"go1.25"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	version, err := client.Version()
	if err != nil {
		t.Fatalf("Version() error = %v", err)
	}
	if version.Version != "1.2.3" || version.Commit != "abc123" {
		t.Fatalf("unexpected version response: %#v", version)
	}
}
