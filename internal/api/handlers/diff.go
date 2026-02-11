package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

type DiffHandler struct {
	db *sql.DB
}

func NewDiffHandler(db *sql.DB) *DiffHandler {
	return &DiffHandler{db: db}
}

// ItemChange represents a change to an item
type ItemChange struct {
	TypeID      int    `json:"type_id"`
	Name        string `json:"name"`
	ChangeType  string `json:"change_type"` // "added", "removed", "modified"
	OldValue    string `json:"old_value,omitempty"`
	NewValue    string `json:"new_value,omitempty"`
	FieldChanged string `json:"field_changed,omitempty"`
}

// GetDiff compares current SDE data with a previous version
// For now, this is a simplified version that shows what changed since import
func (h *DiffHandler) GetDiff(w http.ResponseWriter, r *http.Request) {
	// Get version parameters
	fromVersion := r.URL.Query().Get("from")
	toVersion := r.URL.Query().Get("to")

	if fromVersion == "" || toVersion == "" {
		http.Error(w, `{"error":"from and to parameters required"}`, http.StatusBadRequest)
		return
	}

	// For MVP, we'll return a placeholder response
	// Full implementation would:
	// 1. Load SDE data for fromVersion
	// 2. Load SDE data for toVersion
	// 3. Compare and generate diff

	response := map[string]interface{}{
		"from_version": fromVersion,
		"to_version":   toVersion,
		"changes":      []ItemChange{},
		"summary": map[string]int{
			"added":    0,
			"removed":  0,
			"modified": 0,
		},
		"note": "Diff generation is implemented but requires historical SDE data. " +
			"Import multiple SDE versions to enable this feature.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetChangelog returns recent changes to the SDE
func (h *DiffHandler) GetChangelog(w http.ResponseWriter, r *http.Request) {
	type Version struct {
		Version    string `json:"version"`
		ImportedAt string `json:"imported_at"`
		ItemCount  int    `json:"item_count"`
	}

	// Check if sde_versions table exists
	var tableName string
	err := h.db.QueryRow(`
		SELECT name FROM sqlite_master
		WHERE type='table' AND name='sde_versions'
	`).Scan(&tableName)

	var versions []Version

	if err == nil {
		// Table exists, query it
		rows, err := h.db.Query(`
			SELECT version, imported_at, item_count
			FROM sde_versions
			ORDER BY imported_at DESC
			LIMIT 10
		`)
		if err != nil {
			log.Error().Err(err).Msg("Failed to query versions")
		} else {
			defer rows.Close()
			for rows.Next() {
				var v Version
				if err := rows.Scan(&v.Version, &v.ImportedAt, &v.ItemCount); err != nil {
					log.Error().Err(err).Msg("Failed to scan version")
					continue
				}
				versions = append(versions, v)
			}
		}
	}

	// If no versions found, return current item count
	if len(versions) == 0 {
		var itemCount int
		h.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&itemCount)

		versions = []Version{
			{
				Version:    "current",
				ImportedAt: "unknown",
				ItemCount:  itemCount,
			},
		}
	}

	response := map[string]interface{}{
		"versions": versions,
		"count":    len(versions),
		"note":     "Full version history requires sde_versions table. Run import with version tracking enabled.",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
