package graphql

import (
	"database/sql"
	"regexp"

	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog/log"
)

// Item represents an EVE Online item
type Item struct {
	TypeID      int     `json:"typeId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Volume      float64 `json:"volume"`
}

// resolveItem resolves a single item by ID
func resolveItem(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, nil
	}

	var item Item
	err := db.QueryRow(`
		SELECT type_id, name, description, volume
		FROM items WHERE type_id = ?
	`, id).Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("GraphQL: failed to fetch item")
		return nil, err
	}

	return item, nil
}

// resolveItems resolves a list of items with pagination
func resolveItems(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	limit := p.Args["limit"].(int)
	offset := p.Args["offset"].(int)

	// Validate limits
	if limit > 100 {
		limit = 100
	}
	if limit < 1 {
		limit = 50
	}

	rows, err := db.Query(`
		SELECT type_id, name, description, volume
		FROM items
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("GraphQL: failed to fetch items")
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume); err != nil {
			log.Warn().Err(err).Msg("GraphQL: failed to scan item")
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// resolveSearch resolves item search
func resolveSearch(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	query, ok := p.Args["query"].(string)
	if !ok || query == "" {
		return []Item{}, nil
	}

	limit := p.Args["limit"].(int)
	if limit > 100 {
		limit = 100
	}

	// Sanitize query
	query = sanitizeQuery(query)

	rows, err := db.Query(`
		SELECT i.type_id, i.name, i.description, i.volume
		FROM items_fts fts
		JOIN items i ON i.type_id = fts.type_id
		WHERE items_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		log.Error().Err(err).Str("query", query).Msg("GraphQL: search failed")
		return nil, err
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume); err != nil {
			log.Warn().Err(err).Msg("GraphQL: failed to scan item")
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// sanitizeQuery removes dangerous FTS5 operators
func sanitizeQuery(query string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`)
	return re.ReplaceAllString(query, " ")
}
