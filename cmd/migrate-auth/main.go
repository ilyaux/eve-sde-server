package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "./data/sde.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create api_keys table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			rate_limit INTEGER NOT NULL DEFAULT 60,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			active BOOLEAN NOT NULL DEFAULT 1
		);
		CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key);
		CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active);
	`)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("✓ API keys table created")

	// Create a default API key for testing
	_, err = db.Exec(`
		INSERT OR IGNORE INTO api_keys (key, name, rate_limit, active)
		VALUES ('esk_test_key_for_development_only', 'Development Key', 1000, 1)
	`)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("✓ Development API key created: esk_test_key_for_development_only")
	log.Println("✓ Migration complete!")
}
