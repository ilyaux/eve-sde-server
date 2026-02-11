package database

import (
	"database/sql"
	"time"

	"github.com/rs/zerolog/log"
	_ "modernc.org/sqlite"
)

func New(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Configure connection pool for optimal performance
	db.SetMaxOpenConns(25)                  // Maximum 25 concurrent connections
	db.SetMaxIdleConns(5)                   // Keep 5 idle connections ready
	db.SetConnMaxLifetime(5 * time.Minute)  // Recycle connections every 5 minutes
	db.SetConnMaxIdleTime(1 * time.Minute)  // Close idle connections after 1 minute

	// Enable WAL mode for better concurrency
	_, err = db.Exec("PRAGMA journal_mode=WAL;")
	if err != nil {
		return nil, err
	}

	// Additional SQLite optimizations
	_, err = db.Exec(`
		PRAGMA synchronous=NORMAL;
		PRAGMA cache_size=-64000;
		PRAGMA temp_store=MEMORY;
		PRAGMA busy_timeout=5000;
	`)
	if err != nil {
		return nil, err
	}

	log.Info().
		Int("max_open_conns", 25).
		Int("max_idle_conns", 5).
		Msg("database connection pool configured")

	return db, nil
}
