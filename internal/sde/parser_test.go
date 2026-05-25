package sde

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestParserAndImporter(t *testing.T) {
	root := writeSDEFixture(t)

	parser := NewParser(root)
	types, err := parser.ParseTypes()
	if err != nil {
		t.Fatalf("ParseTypes() error = %v", err)
	}
	if len(types) != 2 {
		t.Fatalf("expected 2 types, got %d", len(types))
	}
	if types[0].ID != 34 || types[0].Name != "Tritanium" || types[0].GroupID != 18 {
		t.Fatalf("unexpected first type: %+v", types[0])
	}

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	importer := NewImporter(db)
	if err := importer.ImportAll(parser); err != nil {
		t.Fatalf("ImportAll() error = %v", err)
	}

	var itemCount int
	if err := db.QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount); err != nil {
		t.Fatalf("count items: %v", err)
	}
	if itemCount != 2 {
		t.Fatalf("expected 2 imported items, got %d", itemCount)
	}

	var categoryID int
	if err := db.QueryRow("SELECT category_id FROM items WHERE type_id = 34").Scan(&categoryID); err != nil {
		t.Fatalf("query imported item: %v", err)
	}
	if categoryID != 4 {
		t.Fatalf("expected category 4, got %d", categoryID)
	}

	var name string
	if err := db.QueryRow(`
		SELECT i.name
		FROM items_fts fts
		JOIN items i ON i.type_id = fts.type_id
		WHERE items_fts MATCH ?
	`, "mineral").Scan(&name); err != nil {
		t.Fatalf("FTS search failed: %v", err)
	}
	if name == "" {
		t.Fatal("expected FTS result")
	}
}

func writeSDEFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	fsd := filepath.Join(root, "sde", "fsd")
	if err := os.MkdirAll(fsd, 0755); err != nil {
		t.Fatalf("create fixture dirs: %v", err)
	}

	writeFixtureFile(t, filepath.Join(fsd, "categories.yaml"), `
4:
  name:
    en: Material
  published: true
`)
	writeFixtureFile(t, filepath.Join(fsd, "groups.yaml"), `
18:
  categoryID: 4
  name:
    en: Mineral
  published: true
`)
	writeFixtureFile(t, filepath.Join(fsd, "types.yaml"), `
34:
  groupID: 18
  name:
    en: Tritanium
  description:
    en: A common mineral.
  volume: 0.01
  published: true
35:
  groupID: 18
  name:
    en: Pyerite
  description:
    en: Another mineral.
  volume: 0.01
  published: true
`)

	return root
}

func writeFixtureFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatalf("write fixture %s: %v", path, err)
	}
}
