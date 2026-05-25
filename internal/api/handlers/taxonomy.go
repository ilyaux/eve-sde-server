package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"
)

type Category struct {
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Published  bool   `json:"published"`
}

type Group struct {
	GroupID    int    `json:"group_id"`
	CategoryID int    `json:"category_id"`
	Name       string `json:"name"`
	Published  bool   `json:"published"`
}

type TaxonomyHandler struct {
	db *sql.DB
}

func NewTaxonomyHandler(db *sql.DB) *TaxonomyHandler {
	return &TaxonomyHandler{db: db}
}

func (h *TaxonomyHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r, 50, 200)

	var total int
	if err := h.db.QueryRow("SELECT COUNT(*) FROM categories").Scan(&total); err != nil {
		log.Error().Err(err).Msg("failed to count categories")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	rows, err := h.db.Query(`
		SELECT category_id, name, published
		FROM categories
		ORDER BY category_id
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to list categories")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	categories := make([]Category, 0)
	for rows.Next() {
		var category Category
		if err := rows.Scan(&category.CategoryID, &category.Name, &category.Published); err != nil {
			log.Warn().Err(err).Msg("failed to scan category")
			continue
		}
		categories = append(categories, category)
	}

	writePagedJSON(w, categories, len(categories), total, limit, offset)
}

func (h *TaxonomyHandler) GetCategory(w http.ResponseWriter, r *http.Request) {
	id, ok := positiveIntParam(w, r, "id", "Invalid category ID - must be a positive integer")
	if !ok {
		return
	}

	var category Category
	err := h.db.QueryRow(`
		SELECT category_id, name, published
		FROM categories
		WHERE category_id = ?
	`, id).Scan(&category.CategoryID, &category.Name, &category.Published)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Category not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error().Err(err).Int("category_id", id).Msg("failed to fetch category")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, category)
}

func (h *TaxonomyHandler) ListGroups(w http.ResponseWriter, r *http.Request) {
	limit, offset := parsePagination(r, 50, 500)
	categoryID, hasCategoryFilter, ok := optionalPositiveIntQuery(w, r, "category_id", "Invalid category_id - must be a positive integer")
	if !ok {
		return
	}

	h.listGroups(w, limit, offset, categoryID, hasCategoryFilter)
}

func (h *TaxonomyHandler) ListCategoryGroups(w http.ResponseWriter, r *http.Request) {
	categoryID, ok := positiveIntParam(w, r, "id", "Invalid category ID - must be a positive integer")
	if !ok {
		return
	}

	limit, offset := parsePagination(r, 50, 500)
	h.listGroups(w, limit, offset, categoryID, true)
}

func (h *TaxonomyHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id, ok := positiveIntParam(w, r, "id", "Invalid group ID - must be a positive integer")
	if !ok {
		return
	}

	var group Group
	err := h.db.QueryRow(`
		SELECT group_id, category_id, name, published
		FROM groups
		WHERE group_id = ?
	`, id).Scan(&group.GroupID, &group.CategoryID, &group.Name, &group.Published)
	if err == sql.ErrNoRows {
		http.Error(w, `{"error":"Group not found"}`, http.StatusNotFound)
		return
	}
	if err != nil {
		log.Error().Err(err).Int("group_id", id).Msg("failed to fetch group")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	writeJSON(w, group)
}

func (h *TaxonomyHandler) listGroups(w http.ResponseWriter, limit, offset, categoryID int, filterByCategory bool) {
	countQuery := "SELECT COUNT(*) FROM groups"
	selectQuery := `
		SELECT group_id, category_id, name, published
		FROM groups
		ORDER BY group_id
		LIMIT ? OFFSET ?
	`
	args := []interface{}{limit, offset}
	countArgs := []interface{}{}

	if filterByCategory {
		countQuery = "SELECT COUNT(*) FROM groups WHERE category_id = ?"
		selectQuery = `
			SELECT group_id, category_id, name, published
			FROM groups
			WHERE category_id = ?
			ORDER BY group_id
			LIMIT ? OFFSET ?
		`
		countArgs = append(countArgs, categoryID)
		args = []interface{}{categoryID, limit, offset}
	}

	var total int
	if err := h.db.QueryRow(countQuery, countArgs...).Scan(&total); err != nil {
		log.Error().Err(err).Msg("failed to count groups")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}

	rows, err := h.db.Query(selectQuery, args...)
	if err != nil {
		log.Error().Err(err).Msg("failed to list groups")
		http.Error(w, `{"error":"Internal server error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	groups := make([]Group, 0)
	for rows.Next() {
		var group Group
		if err := rows.Scan(&group.GroupID, &group.CategoryID, &group.Name, &group.Published); err != nil {
			log.Warn().Err(err).Msg("failed to scan group")
			continue
		}
		groups = append(groups, group)
	}

	writePagedJSON(w, groups, len(groups), total, limit, offset)
}

func writePagedJSON(w http.ResponseWriter, data interface{}, count, total, limit, offset int) {
	writeJSON(w, map[string]interface{}{
		"data": data,
		"meta": map[string]int{
			"count":  count,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func positiveIntParam(w http.ResponseWriter, r *http.Request, name, message string) (int, bool) {
	id, err := strconv.Atoi(chi.URLParam(r, name))
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"`+message+`"}`, http.StatusBadRequest)
		return 0, false
	}

	return id, true
}

func optionalPositiveIntQuery(w http.ResponseWriter, r *http.Request, name, message string) (int, bool, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0, false, true
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		http.Error(w, `{"error":"`+message+`"}`, http.StatusBadRequest)
		return 0, false, false
	}

	return value, true, true
}
