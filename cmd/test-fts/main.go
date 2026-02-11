package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "data/sde.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Test FTS5 search
	rows, err := db.Query(`
		SELECT i.type_id, i.name
		FROM items_fts fts
		JOIN items i ON i.type_id = fts.type_id
		WHERE items_fts MATCH ?
		LIMIT 5
	`, "frigate")

	if err != nil {
		log.Fatal("FTS query failed:", err)
	}
	defer rows.Close()

	fmt.Println("Search results for 'frigate':")
	for rows.Next() {
		var id int
		var name string
		rows.Scan(&id, &name)
		fmt.Printf("%d: %s\n", id, name)
	}
}
