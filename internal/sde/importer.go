package sde

import (
	"database/sql"
	"fmt"
)

// Importer writes parsed SDE data into SQLite.
type Importer struct {
	db *sql.DB
}

// NewImporter creates an SDE importer.
func NewImporter(db *sql.DB) *Importer {
	return &Importer{db: db}
}

// ImportAll imports categories, groups, types, and rebuilds the FTS index.
func (i *Importer) ImportAll(parser *Parser) error {
	if i == nil || i.db == nil {
		return fmt.Errorf("importer has no database")
	}
	if parser == nil {
		return fmt.Errorf("parser is nil")
	}

	categories, err := parser.ParseCategories()
	if err != nil {
		return err
	}
	groups, err := parser.ParseGroups()
	if err != nil {
		return err
	}
	types, err := parser.ParseTypes()
	if err != nil {
		return err
	}

	if err := i.ensureSchema(); err != nil {
		return err
	}

	tx, err := i.db.Begin()
	if err != nil {
		return fmt.Errorf("begin import transaction: %w", err)
	}
	defer tx.Rollback()

	if err := execAll(tx,
		"DROP TRIGGER IF EXISTS items_ai",
		"DROP TRIGGER IF EXISTS items_ad",
		"DROP TRIGGER IF EXISTS items_au",
		"DROP TABLE IF EXISTS items_fts",
		"DELETE FROM items",
		"DELETE FROM groups",
		"DELETE FROM categories",
	); err != nil {
		return err
	}

	if err := insertCategories(tx, categories); err != nil {
		return err
	}
	if err := insertGroups(tx, groups); err != nil {
		return err
	}
	if err := insertTypes(tx, groups, types); err != nil {
		return err
	}
	if err := createFTS(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit import transaction: %w", err)
	}

	return nil
}

func (i *Importer) ensureSchema() error {
	if err := execAll(i.db,
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
		`CREATE INDEX IF NOT EXISTS idx_groups_category_id ON groups(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_group_id ON items(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_items_category_id ON items(category_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sde_versions_checksum ON sde_versions(checksum)`,
		`CREATE INDEX IF NOT EXISTS idx_sde_versions_downloaded_at ON sde_versions(downloaded_at DESC)`,
	); err != nil {
		return err
	}

	columns := []struct {
		table      string
		column     string
		definition string
	}{
		{"items", "group_id", "group_id INTEGER"},
		{"items", "category_id", "category_id INTEGER"},
		{"items", "published", "published BOOLEAN NOT NULL DEFAULT 1"},
		{"groups", "published", "published BOOLEAN NOT NULL DEFAULT 1"},
		{"categories", "published", "published BOOLEAN NOT NULL DEFAULT 1"},
		{"sde_versions", "imported_at", "imported_at TIMESTAMP"},
		{"sde_versions", "import_duration_seconds", "import_duration_seconds INTEGER"},
		{"sde_versions", "items_count", "items_count INTEGER"},
		{"sde_versions", "error", "error TEXT"},
	}
	for _, column := range columns {
		if err := ensureColumn(i.db, column.table, column.column, column.definition); err != nil {
			return err
		}
	}

	return nil
}

type execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func execAll(db execer, statements ...string) error {
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("exec %q: %w", statement, err)
		}
	}
	return nil
}

func ensureColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return fmt.Errorf("inspect table %s: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan table info for %s: %w", table, err)
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info for %s: %w", table, err)
	}

	if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", table, column, err)
	}

	return nil
}

func insertCategories(tx *sql.Tx, categories []Category) error {
	stmt, err := tx.Prepare(`
		INSERT INTO categories (category_id, name, published)
		VALUES (?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare categories insert: %w", err)
	}
	defer stmt.Close()

	for _, category := range categories {
		if _, err := stmt.Exec(category.ID, category.Name, category.Published); err != nil {
			return fmt.Errorf("insert category %d: %w", category.ID, err)
		}
	}

	return nil
}

func insertGroups(tx *sql.Tx, groups []Group) error {
	stmt, err := tx.Prepare(`
		INSERT INTO groups (group_id, category_id, name, published)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare groups insert: %w", err)
	}
	defer stmt.Close()

	for _, group := range groups {
		if _, err := stmt.Exec(group.ID, group.CategoryID, group.Name, group.Published); err != nil {
			return fmt.Errorf("insert group %d: %w", group.ID, err)
		}
	}

	return nil
}

func insertTypes(tx *sql.Tx, groups []Group, types []Type) error {
	groupCategories := make(map[int]int, len(groups))
	for _, group := range groups {
		groupCategories[group.ID] = group.CategoryID
	}

	stmt, err := tx.Prepare(`
		INSERT INTO items (type_id, name, description, volume, group_id, category_id, published)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare items insert: %w", err)
	}
	defer stmt.Close()

	for _, item := range types {
		categoryID := groupCategories[item.GroupID]
		if _, err := stmt.Exec(item.ID, item.Name, item.Description, item.Volume, item.GroupID, categoryID, item.Published); err != nil {
			return fmt.Errorf("insert type %d: %w", item.ID, err)
		}
	}

	return nil
}

func createFTS(tx *sql.Tx) error {
	return execAll(tx,
		`CREATE VIRTUAL TABLE items_fts USING fts5(
			type_id UNINDEXED,
			name,
			description,
			content=items,
			content_rowid=type_id
		)`,
		`INSERT INTO items_fts(rowid, type_id, name, description)
			SELECT type_id, type_id, name, description FROM items`,
		`CREATE TRIGGER items_ai AFTER INSERT ON items BEGIN
			INSERT INTO items_fts(rowid, type_id, name, description)
			VALUES (new.type_id, new.type_id, new.name, new.description);
		END`,
		`CREATE TRIGGER items_ad AFTER DELETE ON items BEGIN
			INSERT INTO items_fts(items_fts, rowid, type_id, name, description)
			VALUES ('delete', old.type_id, old.type_id, old.name, old.description);
		END`,
		`CREATE TRIGGER items_au AFTER UPDATE ON items BEGIN
			INSERT INTO items_fts(items_fts, rowid, type_id, name, description)
			VALUES ('delete', old.type_id, old.type_id, old.name, old.description);
			INSERT INTO items_fts(rowid, type_id, name, description)
			VALUES (new.type_id, new.type_id, new.name, new.description);
		END`,
	)
}
