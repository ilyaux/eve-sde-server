package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ilya/eve-sde-server/internal/esi"
	"github.com/rs/zerolog/log"
)

type ESIHandler struct {
	client *esi.Client
	cache  map[string]cacheEntry // Simple in-memory cache
}

type cacheEntry struct {
	data      []byte
	expiresAt time.Time
}

func NewESIHandler() *ESIHandler {
	return &ESIHandler{
		client: esi.NewClient(),
		cache:  make(map[string]cacheEntry),
	}
}

// Proxy forwards requests to ESI with caching
func (h *ESIHandler) Proxy(w http.ResponseWriter, r *http.Request) {
	// Extract the ESI path from the request
	// Request comes as /api/esi/universe/types/34/
	// We need to extract /universe/types/34/
	path := strings.TrimPrefix(r.URL.Path, "/api/esi")
	if r.URL.RawQuery != "" {
		path += "?" + r.URL.RawQuery
	}

	// Check cache
	if entry, ok := h.cache[path]; ok && time.Now().Before(entry.expiresAt) {
		log.Debug().Str("path", path).Msg("ESI cache hit")
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(entry.data)
		return
	}

	// Fetch from ESI
	log.Info().Str("path", path).Msg("Proxying to ESI")
	data, err := h.client.Get(path)
	if err != nil {
		log.Error().Err(err).Str("path", path).Msg("ESI proxy failed")
		http.Error(w, `{"error":"ESI request failed"}`, http.StatusBadGateway)
		return
	}

	// Cache for 5 minutes
	h.cache[path] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("X-ESI-Proxy", "eve-sde-server")
	w.Write(data)
}

// GetTypeInfo fetches type info from ESI
func (h *ESIHandler) GetTypeInfo(w http.ResponseWriter, r *http.Request) {
	typeIDStr := chi.URLParam(r, "id")
	typeID, err := strconv.Atoi(typeIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid type id"}`, http.StatusBadRequest)
		return
	}

	cacheKey := "type_" + typeIDStr
	if entry, ok := h.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(entry.data)
		return
	}

	info, err := h.client.GetTypeInfo(typeID)
	if err != nil {
		log.Error().Err(err).Int("type_id", typeID).Msg("Failed to fetch type info")
		http.Error(w, `{"error":"failed to fetch type info from ESI"}`, http.StatusBadGateway)
		return
	}

	data, _ := json.Marshal(info)
	h.cache[cacheKey] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(1 * time.Hour), // Type info rarely changes
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

// GetMarketPrices fetches current market prices
func (h *ESIHandler) GetMarketPrices(w http.ResponseWriter, r *http.Request) {
	cacheKey := "market_prices"
	if entry, ok := h.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(entry.data)
		return
	}

	prices, err := h.client.GetMarketPrices()
	if err != nil {
		log.Error().Err(err).Msg("Failed to fetch market prices")
		http.Error(w, `{"error":"failed to fetch market prices from ESI"}`, http.StatusBadGateway)
		return
	}

	data, _ := json.Marshal(prices)
	h.cache[cacheKey] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(10 * time.Minute), // Prices update frequently
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

// GetMarketHistory fetches market history
func (h *ESIHandler) GetMarketHistory(w http.ResponseWriter, r *http.Request) {
	regionIDStr := chi.URLParam(r, "regionID")
	typeIDStr := chi.URLParam(r, "typeID")

	regionID, err := strconv.Atoi(regionIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid region id"}`, http.StatusBadRequest)
		return
	}

	typeID, err := strconv.Atoi(typeIDStr)
	if err != nil {
		http.Error(w, `{"error":"invalid type id"}`, http.StatusBadRequest)
		return
	}

	cacheKey := "market_history_" + regionIDStr + "_" + typeIDStr
	if entry, ok := h.cache[cacheKey]; ok && time.Now().Before(entry.expiresAt) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.Write(entry.data)
		return
	}

	history, err := h.client.GetMarketHistory(regionID, typeID)
	if err != nil {
		log.Error().Err(err).Int("region_id", regionID).Int("type_id", typeID).Msg("Failed to fetch market history")
		http.Error(w, `{"error":"failed to fetch market history from ESI"}`, http.StatusBadGateway)
		return
	}

	data, _ := json.Marshal(history)
	h.cache[cacheKey] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(1 * time.Hour), // History updates daily
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.Write(data)
}

// ClearCache clears the ESI cache
func (h *ESIHandler) ClearCache(w http.ResponseWriter, r *http.Request) {
	count := len(h.cache)
	h.cache = make(map[string]cacheEntry)
	log.Info().Int("entries", count).Msg("ESI cache cleared")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"cleared": count,
	})
}
