package graphql

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/graphql-go/graphql"
	"github.com/rs/zerolog/log"
)

// Item represents an EVE Online item
type Item struct {
	TypeID      int     `json:"typeId"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Volume      float64 `json:"volume"`
	GroupID     int     `json:"groupId"`
	CategoryID  int     `json:"categoryId"`
	Published   bool    `json:"published"`
}

// Category represents an EVE inventory category.
type Category struct {
	CategoryID int    `json:"categoryId"`
	Name       string `json:"name"`
	Published  bool   `json:"published"`
}

// Group represents an EVE inventory group.
type Group struct {
	GroupID    int    `json:"groupId"`
	CategoryID int    `json:"categoryId"`
	Name       string `json:"name"`
	Published  bool   `json:"published"`
}

// resolveItem resolves a single item by ID
func resolveItem(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, nil
	}

	var item Item
	err := db.QueryRow(`
		SELECT type_id, name, description, volume, COALESCE(group_id, 0), COALESCE(category_id, 0), COALESCE(published, 1)
		FROM items WHERE type_id = ?
	`, id).Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume, &item.GroupID, &item.CategoryID, &item.Published)

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
		SELECT type_id, name, description, volume, COALESCE(group_id, 0), COALESCE(category_id, 0), COALESCE(published, 1)
		FROM items
		ORDER BY type_id
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
		if err := rows.Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume, &item.GroupID, &item.CategoryID, &item.Published); err != nil {
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
	if strings.TrimSpace(query) == "" {
		return []Item{}, nil
	}

	rows, err := db.Query(`
		SELECT i.type_id, i.name, i.description, i.volume, COALESCE(i.group_id, 0), COALESCE(i.category_id, 0), COALESCE(i.published, 1)
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
		if err := rows.Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume, &item.GroupID, &item.CategoryID, &item.Published); err != nil {
			log.Warn().Err(err).Msg("GraphQL: failed to scan item")
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

func resolveCategory(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, nil
	}

	var category Category
	err := db.QueryRow(`
		SELECT category_id, name, published
		FROM categories
		WHERE category_id = ?
	`, id).Scan(&category.CategoryID, &category.Name, &category.Published)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("GraphQL: failed to fetch category")
		return nil, err
	}

	return category, nil
}

func resolveCategories(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	limit, offset := paginationArgs(p, 50, 200)

	rows, err := db.Query(`
		SELECT category_id, name, published
		FROM categories
		ORDER BY category_id
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("GraphQL: failed to fetch categories")
		return nil, err
	}
	defer rows.Close()

	var categories []Category
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.CategoryID, &category.Name, &category.Published); err != nil {
			log.Warn().Err(err).Msg("GraphQL: failed to scan category")
			continue
		}
		categories = append(categories, category)
	}

	return categories, nil
}

func resolveGroup(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	id, ok := p.Args["id"].(int)
	if !ok {
		return nil, nil
	}

	var group Group
	err := db.QueryRow(`
		SELECT group_id, category_id, name, published
		FROM groups
		WHERE group_id = ?
	`, id).Scan(&group.GroupID, &group.CategoryID, &group.Name, &group.Published)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		log.Error().Err(err).Int("id", id).Msg("GraphQL: failed to fetch group")
		return nil, err
	}

	return group, nil
}

func resolveGroups(db *sql.DB, p graphql.ResolveParams) (interface{}, error) {
	limit, offset := paginationArgs(p, 50, 500)

	query := `
		SELECT group_id, category_id, name, published
		FROM groups
		ORDER BY group_id
		LIMIT ? OFFSET ?
	`
	args := []interface{}{limit, offset}
	if categoryID, ok := p.Args["categoryId"].(int); ok && categoryID > 0 {
		query = `
			SELECT group_id, category_id, name, published
			FROM groups
			WHERE category_id = ?
			ORDER BY group_id
			LIMIT ? OFFSET ?
		`
		args = []interface{}{categoryID, limit, offset}
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		log.Error().Err(err).Msg("GraphQL: failed to fetch groups")
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.GroupID, &group.CategoryID, &group.Name, &group.Published); err != nil {
			log.Warn().Err(err).Msg("GraphQL: failed to scan group")
			continue
		}
		groups = append(groups, group)
	}

	return groups, nil
}

// sanitizeQuery removes dangerous FTS5 operators
func sanitizeQuery(query string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`)
	return re.ReplaceAllString(query, " ")
}

func paginationArgs(p graphql.ResolveParams, defaultLimit, maxLimit int) (int, int) {
	limit := defaultLimit
	offset := 0

	if parsed, ok := p.Args["limit"].(int); ok {
		limit = parsed
	}
	if parsed, ok := p.Args["offset"].(int); ok {
		offset = parsed
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if limit < 1 {
		limit = defaultLimit
	}
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
