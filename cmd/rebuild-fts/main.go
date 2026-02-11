package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/sde.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Dropping old FTS index...")
	_, err = db.Exec("DROP TABLE IF EXISTS items_fts")
	if err != nil {
		log.Fatal("Drop failed:", err)
	}

	log.Println("Creating new FTS index...")
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
		log.Fatal("Create failed:", err)
	}

	log.Println("Populating FTS index...")
	_, err = db.Exec(`
		INSERT INTO items_fts(type_id, name, description)
		SELECT type_id, name, description FROM items;
	`)
	if err != nil {
		log.Fatal("Populate failed:", err)
	}

	// Recreate triggers
	log.Println("Creating triggers...")

	_, err = db.Exec(`
		DROP TRIGGER IF EXISTS items_ai;
		CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
			INSERT INTO items_fts(type_id, name, description)
			VALUES (new.type_id, new.name, new.description);
		END;
	`)
	if err != nil {
		log.Fatal("Insert trigger failed:", err)
	}

	_, err = db.Exec(`
		DROP TRIGGER IF EXISTS items_ad;
		CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
			DELETE FROM items_fts WHERE type_id = old.type_id;
		END;
	`)
	if err != nil {
		log.Fatal("Delete trigger failed:", err)
	}

	_, err = db.Exec(`
		DROP TRIGGER IF EXISTS items_au;
		CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
			UPDATE items_fts SET name = new.name, description = new.description
			WHERE type_id = old.type_id;
		END;
	`)
	if err != nil {
		log.Fatal("Update trigger failed:", err)
	}

	// Verify
	var count int
	db.QueryRow("SELECT COUNT(*) FROM items_fts").Scan(&count)

	log.Printf("✓ FTS index rebuilt successfully! (%d items indexed)\n", count)
}
