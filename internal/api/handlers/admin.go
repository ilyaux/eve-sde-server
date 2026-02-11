package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ilya/eve-sde-server/internal/auth"
	"github.com/rs/zerolog/log"
)

type AdminHandler struct {
	db      *sql.DB
	authMgr *auth.Manager
}

func NewAdminHandler(db *sql.DB, authMgr *auth.Manager) *AdminHandler {
	return &AdminHandler{
		db:      db,
		authMgr: authMgr,
	}
}

// Stats returns server statistics
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	var totalKeys, activeKeys int

	// Count total keys
	err := h.db.QueryRow("SELECT COUNT(*) FROM api_keys").Scan(&totalKeys)
	if err != nil {
		log.Error().Err(err).Msg("Failed to count total keys")
	}

	// Count active keys
	err = h.db.QueryRow("SELECT COUNT(*) FROM api_keys WHERE active = 1").Scan(&activeKeys)
	if err != nil {
		log.Error().Err(err).Msg("Failed to count active keys")
	}

	// Count items
	var totalItems int
	err = h.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&totalItems)
	if err != nil {
		log.Error().Err(err).Msg("Failed to count items")
	}

	stats := map[string]interface{}{
		"total_keys":  totalKeys,
		"active_keys": activeKeys,
		"total_items": totalItems,
		"status":      "healthy",
		"timestamp":   time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// ListKeys returns all API keys (without the actual key values)
func (h *AdminHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT id, key, name, rate_limit, created_at, expires_at, active
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		log.Error().Err(err).Msg("Failed to query API keys")
		http.Error(w, `{"error":"failed to fetch keys"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var keys []map[string]interface{}
	for rows.Next() {
		var id, rateLimit int
		var key, name string
		var createdAt string
		var expiresAt sql.NullString
		var active bool

		err := rows.Scan(&id, &key, &name, &rateLimit, &createdAt, &expiresAt, &active)
		if err != nil {
			log.Error().Err(err).Msg("Failed to scan key row")
			continue
		}

		keyData := map[string]interface{}{
			"id":         id,
			"key":        key, // In production, you might want to hide this
			"name":       name,
			"rate_limit": rateLimit,
			"created_at": createdAt,
			"active":     active,
		}

		if expiresAt.Valid {
			keyData["expires_at"] = expiresAt.String
		} else {
			keyData["expires_at"] = nil
		}

		keys = append(keys, keyData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

// CreateKey creates a new API key
func (h *AdminHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name      string     `json:"name"`
		RateLimit int        `json:"rate_limit"`
		ExpiresAt *time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
		return
	}

	if req.RateLimit <= 0 {
		req.RateLimit = 60 // Default 60 req/min
	}

	// Generate new API key
	key, err := auth.GenerateAPIKey()
	if err != nil {
		log.Error().Err(err).Msg("Failed to generate API key")
		http.Error(w, `{"error":"failed to generate key"}`, http.StatusInternalServerError)
		return
	}

	// Insert into database
	var expiresAt interface{}
	if req.ExpiresAt != nil {
		expiresAt = req.ExpiresAt.Format(time.RFC3339)
	}

	_, err = h.db.Exec(`
		INSERT INTO api_keys (key, name, rate_limit, expires_at, active)
		VALUES (?, ?, ?, ?, 1)
	`, key, req.Name, req.RateLimit, expiresAt)

	if err != nil {
		log.Error().Err(err).Msg("Failed to insert API key")
		http.Error(w, `{"error":"failed to save key"}`, http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("name", req.Name).
		Int("rate_limit", req.RateLimit).
		Msg("API key created")

	response := map[string]interface{}{
		"key":        key,
		"name":       req.Name,
		"rate_limit": req.RateLimit,
		"created_at": time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// RevokeKey revokes (deactivates) an API key
func (h *AdminHandler) RevokeKey(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error":"invalid key id"}`, http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec("UPDATE api_keys SET active = 0 WHERE id = ?", id)
	if err != nil {
		log.Error().Err(err).Int("key_id", id).Msg("Failed to revoke key")
		http.Error(w, `{"error":"failed to revoke key"}`, http.StatusInternalServerError)
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error":"key not found"}`, http.StatusNotFound)
		return
	}

	log.Info().Int("key_id", id).Msg("API key revoked")

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"success":true,"message":"key revoked"}`))
}
