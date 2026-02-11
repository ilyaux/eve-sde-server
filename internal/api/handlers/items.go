package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Item struct {
	TypeID      int     `json:"type_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Volume      float64 `json:"volume"`
}

type ItemHandler struct {
	db *sql.DB
}

func NewItemHandler(db *sql.DB) *ItemHandler {
	return &ItemHandler{db: db}
}

func (h *ItemHandler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")

	// Validate input: must be a positive integer
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"Invalid item ID - must be a positive integer"}`, http.StatusBadRequest)
		return
	}

	var item Item
	err = h.db.QueryRow(`
		SELECT type_id, name, description, volume
		FROM items WHERE type_id = ?
	`, id).Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume)

	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Item not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error().Err(err).Int("type_id", id).Msg("failed to fetch item")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(item)
}

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT type_id, name, description, volume
		FROM items
		LIMIT 50
	`)
	if err != nil {
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume); err != nil {
			continue
		}
		items = append(items, item)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": items,
		"meta": map[string]int{"count": len(items)},
	})
}

func (h *ItemHandler) Search(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, `{"error":"Missing query parameter 'q'"}`, http.StatusBadRequest)
		return
	}

	// Security: Validate query length to prevent DoS
	if len(query) > 200 {
		http.Error(w, `{"error":"Query too long - maximum 200 characters"}`, http.StatusBadRequest)
		return
	}

	// Security: Sanitize FTS query to prevent injection
	query = sanitizeFTSQuery(query)

	// Parse pagination parameters
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	// Add timeout to prevent long-running queries
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	rows, err := h.db.QueryContext(ctx, `
		SELECT i.type_id, i.name, i.description, i.volume
		FROM items_fts fts
		JOIN items i ON i.type_id = fts.type_id
		WHERE items_fts MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?
	`, query, limit, offset)
	if err != nil {
		log.Error().Err(err).Str("query", query).Msg("search failed")
		http.Error(w, `{"error":"Search failed"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var items []Item
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.TypeID, &item.Name, &item.Description, &item.Volume); err != nil {
			log.Warn().Err(err).Msg("failed to scan item")
			continue
		}
		items = append(items, item)
	}

	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		http.Error(w, `{"error":"Search timeout - please refine your query"}`, http.StatusRequestTimeout)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": items,
		"meta": map[string]interface{}{
			"count":  len(items),
			"limit":  limit,
			"offset": offset,
		},
	})
}

// sanitizeFTSQuery removes potentially dangerous FTS5 operators
func sanitizeFTSQuery(query string) string {
	// Remove special FTS5 operators that could cause issues
	// Allow only alphanumeric, spaces, and basic punctuation
	re := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`)
	return re.ReplaceAllString(query, " ")
}
