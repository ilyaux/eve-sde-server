package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// VersionInfo describes the running server build.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

type SystemHandler struct {
	db        *sql.DB
	version   VersionInfo
	startedAt time.Time
}

func NewSystemHandler(db *sql.DB, version VersionInfo, startedAt time.Time) *SystemHandler {
	return &SystemHandler{
		db:        db,
		version:   version,
		startedAt: startedAt,
	}
}

func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeSystemJSON(w, http.StatusOK, map[string]interface{}{
		"status":         "OK",
		"uptime_seconds": int64(time.Since(h.startedAt).Seconds()),
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *SystemHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	statusCode := http.StatusOK
	status := "ready"
	checks := map[string]string{}

	if err := h.db.PingContext(ctx); err != nil {
		statusCode = http.StatusServiceUnavailable
		status = "not_ready"
		checks["database"] = "unhealthy"
	} else {
		checks["database"] = "ok"
	}

	var itemsCount int
	if err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM items").Scan(&itemsCount); err != nil {
		statusCode = http.StatusServiceUnavailable
		status = "not_ready"
		checks["items"] = "unavailable"
	} else {
		checks["items"] = "ok"
	}

	var latestVersion sql.NullString
	var latestImportedAt sql.NullString
	err := h.db.QueryRowContext(ctx, `
		SELECT version, imported_at
		FROM sde_versions
		ORDER BY COALESCE(imported_at, downloaded_at) DESC
		LIMIT 1
	`).Scan(&latestVersion, &latestImportedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		checks["sde_version"] = "not_imported"
	case err != nil:
		statusCode = http.StatusServiceUnavailable
		status = "not_ready"
		checks["sde_version"] = "unavailable"
	default:
		checks["sde_version"] = "ok"
	}

	response := map[string]interface{}{
		"status":      status,
		"checks":      checks,
		"items_count": itemsCount,
		"version":     h.version,
	}
	if latestVersion.Valid {
		response["latest_sde_version"] = latestVersion.String
	} else {
		response["latest_sde_version"] = nil
	}
	if latestImportedAt.Valid {
		response["latest_sde_imported_at"] = latestImportedAt.String
	} else {
		response["latest_sde_imported_at"] = nil
	}

	writeSystemJSON(w, statusCode, response)
}

func (h *SystemHandler) Version(w http.ResponseWriter, r *http.Request) {
	writeSystemJSON(w, http.StatusOK, h.version)
}

func writeSystemJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
