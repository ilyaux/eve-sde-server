package main

import (
	"database/sql"
	"log"
	"os"

	_ "modernc.org/sqlite"
)

func main() {
	// Create data directory
	os.MkdirAll("data", 0755)

	// Open database
	db, err := sql.Open("sqlite", "data/sde.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create sde_versions table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS sde_versions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			version TEXT NOT NULL,
			checksum TEXT NOT NULL UNIQUE,
			downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`)
	if err != nil {
		log.Fatal("Failed to create sde_versions table:", err)
	}

	// Create categories table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			category_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			published BOOLEAN DEFAULT 1
		);
	`)
	if err != nil {
		log.Fatal("Failed to create categories table:", err)
	}

	// Create groups table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			group_id INTEGER PRIMARY KEY,
			category_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			published BOOLEAN DEFAULT 1,
			FOREIGN KEY (category_id) REFERENCES categories(category_id)
		);
	`)
	if err != nil {
		log.Fatal("Failed to create groups table:", err)
	}

	// Create items table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			type_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			volume REAL
		);
	`)
	if err != nil {
		log.Fatal("Failed to create items table:", err)
	}

	// Create FTS5 table
	_, err = db.Exec(`
		CREATE VIRTUAL TABLE items_fts USING fts5(
			type_id UNINDEXED,
			name,
			description,
			content=items,
			content_rowid=type_id
		);
	`)
	if err != nil {
		log.Fatal("Failed to create FTS table:", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO items VALUES
			(34, 'Tritanium', 'A heavy, silver-gray metal. Tritanium is the primary building material for most structures and ships in New Eden.', 0.01),
			(35, 'Pyerite', 'A fairly common ore that is very similar to Mexallon in its chemical composition and properties.', 0.01),
			(36, 'Mexallon', 'Malleable precious metal with a high melting point and excellent corrosion resistance.', 0.01),
			(37, 'Isogen', 'Uniquely colored silvery metal. Isogen is considered one of the most important minerals in the universe.', 0.01),
			(38, 'Nocxium', 'A very rare mineral that possesses unique physical and chemical properties.', 0.01),
			(39, 'Zydrine', 'Highly valued ore, with distinctive greenish hue. Zydrine is second only to Megacyte in rarity.', 0.01),
			(40, 'Megacyte', 'The rarest of ores. Megacyte is used extensively in the construction of capital ships.', 0.01),
			(587, 'Rifter', 'The Rifter is a very powerful combat frigate and can easily tackle the best frigates out there.', 24850.0);
	`)
	if err != nil {
		log.Fatal("Failed to insert data:", err)
	}

	// Populate FTS
	_, err = db.Exec(`
		INSERT INTO items_fts(type_id, name, description)
		SELECT type_id, name, description FROM items;
	`)
	if err != nil {
		log.Fatal("Failed to populate FTS:", err)
	}

	// Create triggers
	_, err = db.Exec(`
		CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
			INSERT INTO items_fts(type_id, name, description)
			VALUES (new.type_id, new.name, new.description);
		END;
	`)
	if err != nil {
		log.Fatal("Failed to create insert trigger:", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
			DELETE FROM items_fts WHERE type_id = old.type_id;
		END;
	`)
	if err != nil {
		log.Fatal("Failed to create delete trigger:", err)
	}

	_, err = db.Exec(`
		CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
			UPDATE items_fts SET name = new.name, description = new.description
			WHERE type_id = old.type_id;
		END;
	`)
	if err != nil {
		log.Fatal("Failed to create update trigger:", err)
	}

	log.Println("✓ Database created successfully!")
	log.Println("✓ Test data inserted (8 items)")

	// Verify
	var count int
	db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count)
	log.Printf("✓ Database contains %d items\n", count)
}
