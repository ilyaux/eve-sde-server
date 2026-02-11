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

	// Add categories table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS categories (
			category_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			published BOOLEAN DEFAULT 1
		);
	`)
	if err != nil {
		log.Fatal("categories:", err)
	}

	// Add groups table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS groups (
			group_id INTEGER PRIMARY KEY,
			category_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			published BOOLEAN DEFAULT 1
		);
	`)
	if err != nil {
		log.Fatal("groups:", err)
	}

	log.Println("✓ Tables added successfully!")
}
