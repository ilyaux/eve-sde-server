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

func TestDiff(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/diff" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("from") != "20250101" || r.URL.Query().Get("to") != "20250201" {
			http.Error(w, "unexpected query", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"from_version":"20250101",
			"to_version":"20250201",
			"changes":[{"type_id":34,"name":"Tritanium","change_type":"modified","field_changed":"volume","old_value":"0.01","new_value":"0.02"}],
			"summary":{"added":0,"removed":0,"modified":1}
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	diff, err := client.Diff("20250101", "20250201")
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if diff.FromVersion != "20250101" || diff.ToVersion != "20250201" {
		t.Fatalf("unexpected versions: %#v", diff)
	}
	if diff.Summary.Modified != 1 || len(diff.Changes) != 1 {
		t.Fatalf("unexpected diff response: %#v", diff)
	}
}

func TestChangelog(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/changelog" {
			http.Error(w, "unexpected path", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"versions":[{"version":"20250101","imported_at":"2025-01-01T00:00:00Z","item_count":25000}],
			"count":1,
			"note":"ok"
		}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	changelog, err := client.Changelog()
	if err != nil {
		t.Fatalf("Changelog() error = %v", err)
	}
	if changelog.Count != 1 || len(changelog.Versions) != 1 {
		t.Fatalf("unexpected changelog response: %#v", changelog)
	}
	if changelog.Versions[0].ItemCount != 25000 {
		t.Fatalf("unexpected item count: %#v", changelog.Versions[0])
	}
}

func TestESIHelpers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/esi/types/34":
			w.Write([]byte(`{"type_id":34,"name":"Tritanium"}`))
		case "/api/esi/markets/prices":
			w.Write([]byte(`[{"type_id":34,"average_price":6.1}]`))
		case "/api/esi/markets/10000002/history/34":
			w.Write([]byte(`[{"date":"2026-01-01","average":6.1,"volume":1000}]`))
		default:
			http.Error(w, "unexpected path", http.StatusNotFound)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")

	typeInfo, err := client.ESITypeInfo(34)
	if err != nil {
		t.Fatalf("ESITypeInfo() error = %v", err)
	}
	if typeInfo["name"] != "Tritanium" {
		t.Fatalf("unexpected type info: %#v", typeInfo)
	}

	prices, err := client.ESIMarketPrices()
	if err != nil {
		t.Fatalf("ESIMarketPrices() error = %v", err)
	}
	if len(prices) != 1 || prices[0]["type_id"].(float64) != 34 {
		t.Fatalf("unexpected market prices: %#v", prices)
	}

	history, err := client.ESIMarketHistory(10000002, 34)
	if err != nil {
		t.Fatalf("ESIMarketHistory() error = %v", err)
	}
	if len(history) != 1 || history[0]["date"] != "2026-01-01" {
		t.Fatalf("unexpected market history: %#v", history)
	}
}
