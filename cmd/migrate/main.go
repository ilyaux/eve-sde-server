package main

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite", "data/sde.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	statements := []string{
		`CREATE TABLE IF NOT EXISTS sde_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version TEXT NOT NULL,
			checksum TEXT NOT NULL UNIQUE,
			downloaded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			imported_at TIMESTAMP,
			import_duration_seconds INTEGER,
			items_count INTEGER,
			error TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sde_versions_checksum ON sde_versions(checksum)`,
		`CREATE INDEX IF NOT EXISTS idx_sde_versions_downloaded_at ON sde_versions(downloaded_at DESC)`,
		`CREATE TABLE IF NOT EXISTS categories (
			category_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS groups (
			group_id INTEGER PRIMARY KEY,
			category_id INTEGER,
			name TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT 1,
			FOREIGN KEY (category_id) REFERENCES categories(category_id)
		)`,
		`CREATE TABLE IF NOT EXISTS items (
			type_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			volume REAL,
			group_id INTEGER,
			category_id INTEGER,
			published BOOLEAN NOT NULL DEFAULT 1,
			FOREIGN KEY (group_id) REFERENCES groups(group_id),
			FOREIGN KEY (category_id) REFERENCES categories(category_id)
		)`,
		`CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			key_hash TEXT,
			name TEXT NOT NULL,
			rate_limit INTEGER NOT NULL DEFAULT 60,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			active BOOLEAN NOT NULL DEFAULT 1
		)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_key ON api_keys(key)`,
		`CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(active)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
			type_id UNINDEXED,
			name,
			description,
			content=items,
			content_rowid=type_id
		)`,
		`CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
			INSERT INTO items_fts(rowid, type_id, name, description)
			VALUES (new.type_id, new.type_id, new.name, new.description);
		END`,
		`CREATE TRIGGER IF NOT EXISTS items_ad AFTER DELETE ON items BEGIN
			INSERT INTO items_fts(items_fts, rowid, type_id, name, description)
			VALUES ('delete', old.type_id, old.type_id, old.name, old.description);
		END`,
		`CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
			INSERT INTO items_fts(items_fts, rowid, type_id, name, description)
			VALUES ('delete', old.type_id, old.type_id, old.name, old.description);
			INSERT INTO items_fts(rowid, type_id, name, description)
			VALUES (new.type_id, new.type_id, new.name, new.description);
		END`,
		`INSERT OR IGNORE INTO categories (category_id, name, published) VALUES
			(4, 'Material', 1),
			(6, 'Ship', 1)`,
		`INSERT OR IGNORE INTO groups (group_id, category_id, name, published) VALUES
			(18, 4, 'Mineral', 1),
			(25, 6, 'Frigate', 1)`,
		`INSERT OR IGNORE INTO items (type_id, name, description, volume, group_id, category_id, published) VALUES
			(34, 'Tritanium', 'A heavy, silver-gray metal. Tritanium is the primary building material for most structures and ships in New Eden.', 0.01, 18, 4, 1),
			(35, 'Pyerite', 'A fairly common ore that is very similar to Mexallon in its chemical composition and properties.', 0.01, 18, 4, 1),
			(36, 'Mexallon', 'Malleable precious metal with a high melting point and excellent corrosion resistance.', 0.01, 18, 4, 1),
			(37, 'Isogen', 'Uniquely colored silvery metal. Isogen is considered one of the most important minerals in the universe.', 0.01, 18, 4, 1),
			(38, 'Nocxium', 'A very rare mineral that possesses unique physical and chemical properties.', 0.01, 18, 4, 1),
			(39, 'Zydrine', 'Highly valued ore, with distinctive greenish hue. Zydrine is second only to Megacyte in rarity.', 0.01, 18, 4, 1),
			(40, 'Megacyte', 'The rarest of ores. Megacyte is used extensively in the construction of capital ships.', 0.01, 18, 4, 1),
			(587, 'Rifter', 'The Rifter is a very powerful combat frigate and can easily tackle the best frigates out there.', 24850.0, 25, 6, 1)`,
		`INSERT INTO items_fts(rowid, type_id, name, description)
			SELECT type_id, type_id, name, description
			FROM items
			WHERE type_id NOT IN (SELECT rowid FROM items_fts)`,
	}

	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			log.Fatalf("migration failed: %v\nSQL: %s", err, statement)
		}
	}
	if err := ensureColumn(db, "api_keys", "key_hash", "key_hash TEXT"); err != nil {
		log.Fatal(err)
	}
	if _, err := db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE key_hash IS NOT NULL`); err != nil {
		log.Fatal(err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		log.Fatal(err)
	}

	log.Printf("Database is ready with %d sample items\n", count)
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + definition)
	return err
}
