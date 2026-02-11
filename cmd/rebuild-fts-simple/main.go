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

	log.Println("Creating simple FTS index (without content=)...")
	_, err = db.Exec(`
		CREATE VIRTUAL TABLE items_fts USING fts5(
			type_id,
			name,
			description
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

	// Verify
	var count int
	db.QueryRow("SELECT COUNT(*) FROM items_fts").Scan(&count)

	log.Printf("✓ Simple FTS index created! (%d items indexed)\n", count)

	// Test query
	log.Println("\nTesting search for 'frigate':")
	rows, err := db.Query(`
		SELECT type_id, name
		FROM items_fts
		WHERE items_fts MATCH 'frigate'
		LIMIT 5
	`)
	if err != nil {
		log.Fatal("Search failed:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		log.Printf("  %d: %s\n", id, name)
	}
}
