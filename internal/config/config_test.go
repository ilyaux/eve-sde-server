package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	withCleanEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 8080 {
		t.Fatalf("expected default port 8080, got %d", cfg.Port)
	}
	if cfg.DBPath != "data/sde.db" {
		t.Fatalf("expected default DB path, got %q", cfg.DBPath)
	}
	if cfg.AuthEnabled {
		t.Fatal("expected auth to be disabled by default")
	}
	if cfg.SDEAutoUpdate {
		t.Fatal("expected SDE auto-update to be disabled by default")
	}
	if cfg.SDEDataDir != "data" {
		t.Fatalf("expected default SDE data dir, got %q", cfg.SDEDataDir)
	}
	if cfg.CacheTTL != 60*time.Second {
		t.Fatalf("expected default cache TTL 60s, got %s", cfg.CacheTTL)
	}
	if cfg.CacheMaxSizeMB != 100 {
		t.Fatalf("expected default cache max size 100 MB, got %d", cfg.CacheMaxSizeMB)
	}
}

func TestLoadOverrides(t *testing.T) {
	withCleanEnv(t)
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "tmp/test.db")
	t.Setenv("AUTH_ENABLED", "true")
	t.Setenv("SDE_AUTO_UPDATE", "true")
	t.Setenv("SDE_URL", "https://example.test/sde.zip")
	t.Setenv("SDE_DATA_DIR", "tmp/sde")
	t.Setenv("CACHE_TTL", "5m")
	t.Setenv("CACHE_MAX_SIZE_MB", "512")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Port != 9090 || cfg.DBPath != "tmp/test.db" {
		t.Fatalf("unexpected port/path: %#v", cfg)
	}
	if !cfg.AuthEnabled || !cfg.SDEAutoUpdate {
		t.Fatalf("expected auth and SDE auto-update to be enabled: %#v", cfg)
	}
	if cfg.SDEURL != "https://example.test/sde.zip" || cfg.SDEDataDir != "tmp/sde" {
		t.Fatalf("unexpected SDE config: %#v", cfg)
	}
	if cfg.CacheTTL != 5*time.Minute || cfg.CacheMaxSizeMB != 512 {
		t.Fatalf("unexpected cache config: %#v", cfg)
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "port too low", key: "PORT", value: "0"},
		{name: "port too high", key: "PORT", value: "70000"},
		{name: "cache ttl", key: "CACHE_TTL", value: "0s"},
		{name: "cache max size", key: "CACHE_MAX_SIZE_MB", value: "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withCleanEnv(t)
			t.Setenv(tt.key, tt.value)

			if _, err := Load(); err == nil {
				t.Fatal("expected Load() to reject invalid config")
			}
		})
	}
}

func withCleanEnv(t *testing.T) {
	t.Helper()

	keys := []string{
		"PORT",
		"DB_PATH",
		"TLS_ENABLED",
		"TLS_CERT_FILE",
		"TLS_KEY_FILE",
		"ALLOWED_ORIGINS",
		"AUTH_ENABLED",
		"SDE_AUTO_UPDATE",
		"SDE_URL",
		"SDE_DATA_DIR",
		"CACHE_TTL",
		"CACHE_MAX_SIZE_MB",
	}

	previous := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		previous[key] = value
		present[key] = ok
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
	}

	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				_ = os.Setenv(key, previous[key])
			} else {
				_ = os.Unsetenv(key)
			}
		}
	})
}
