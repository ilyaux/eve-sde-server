package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ilyaux/eve-sde-server/internal/auth"
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
	keys, err := h.authMgr.ListAPIKeys(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("Failed to query API keys")
		http.Error(w, `{"error":"failed to fetch keys"}`, http.StatusInternalServerError)
		return
	}

	response := make([]map[string]interface{}, 0, len(keys))
	for _, key := range keys {
		keyPreview := maskAPIKey(key.Key)

		keyData := map[string]interface{}{
			"id":          key.ID,
			"key":         keyPreview,
			"key_preview": keyPreview,
			"name":        key.Name,
			"rate_limit":  key.RateLimit,
			"created_at":  key.CreatedAt.Format(time.RFC3339),
			"active":      key.Active,
		}

		if key.ExpiresAt != nil {
			keyData["expires_at"] = key.ExpiresAt.Format(time.RFC3339)
		} else {
			keyData["expires_at"] = nil
		}

		response = append(response, keyData)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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

	var expiresIn *time.Duration
	if req.ExpiresAt != nil {
		duration := time.Until(*req.ExpiresAt)
		if duration <= 0 {
			http.Error(w, `{"error":"expires_at must be in the future"}`, http.StatusBadRequest)
			return
		}
		expiresIn = &duration
	}

	apiKey, err := h.authMgr.CreateAPIKey(r.Context(), req.Name, req.RateLimit, expiresIn)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create API key")
		http.Error(w, `{"error":"failed to save key"}`, http.StatusInternalServerError)
		return
	}

	log.Info().
		Str("name", req.Name).
		Int("rate_limit", req.RateLimit).
		Msg("API key created")

	response := map[string]interface{}{
		"key":        apiKey.Key,
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

func maskAPIKey(key string) string {
	if len(key) <= 12 {
		return "********"
	}

	return auth.KeyPreview(key)
}
