package main

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEnsureColumnAddsKeyHashBeforeIndex(t *testing.T) {
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
		)
	`)
	if err != nil {
		t.Fatalf("create old api_keys table: %v", err)
	}

	if err := ensureColumn(db, "api_keys", "key_hash", "key_hash TEXT"); err != nil {
		t.Fatalf("ensureColumn() error = %v", err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE key_hash IS NOT NULL`); err != nil {
		t.Fatalf("create key_hash index: %v", err)
	}

	var found bool
	rows, err := db.Query("PRAGMA table_info(api_keys)")
	if err != nil {
		t.Fatalf("query table info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan table info: %v", err)
		}
		if name == "key_hash" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info: %v", err)
	}
	if !found {
		t.Fatal("key_hash column was not added")
	}
}
