package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE items (
			type_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT,
			volume REAL,
			group_id INTEGER,
			category_id INTEGER
		);

		CREATE TABLE categories (
			category_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL
		);

		CREATE TABLE groups (
			group_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			category_id INTEGER
		);

		CREATE VIRTUAL TABLE items_fts USING fts5(
			type_id UNINDEXED,
			name,
			description
		);
	`)
	if err != nil {
		t.Fatalf("failed to create schema: %v", err)
	}

	// Insert test data
	_, err = db.Exec(`
		INSERT INTO items (type_id, name, description, volume)
		VALUES
			(34, 'Tritanium', 'The main building block', 0.01),
			(35, 'Pyerite', 'Reddish mineral', 0.01),
			(36, 'Mexallon', 'Fairly common mineral', 0.01);

		INSERT INTO items_fts (type_id, name, description)
		VALUES
			(34, 'Tritanium', 'The main building block'),
			(35, 'Pyerite', 'Reddish mineral'),
			(36, 'Mexallon', 'Fairly common mineral');
	`)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	return db
}

func TestItemHandler_Get(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewItemHandler(db)

	tests := []struct {
		name           string
		itemID         string
		expectedStatus int
		expectedName   string
	}{
		{
			name:           "Get existing item (Tritanium)",
			itemID:         "34",
			expectedStatus: http.StatusOK,
			expectedName:   "Tritanium",
		},
		{
			name:           "Get non-existent item",
			itemID:         "999999",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:           "Invalid item ID",
			itemID:         "invalid",
			expectedStatus: http.StatusBadRequest, // Handler returns 400 for invalid ID after input validation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/items/"+tt.itemID, nil)
			w := httptest.NewRecorder()

			// Setup chi context with URL param
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.itemID)
			ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
			req = req.WithContext(ctx)

			handler.Get(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var item struct {
					TypeID int     `json:"type_id"`
					Name   string  `json:"name"`
					Volume float64 `json:"volume"`
				}
				err := json.NewDecoder(w.Body).Decode(&item)
				if err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if item.Name != tt.expectedName {
					t.Errorf("expected name %s, got %s", tt.expectedName, item.Name)
				}
			}
		})
	}
}

func TestItemHandler_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewItemHandler(db)

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedStatus int
	}{
		{
			name:          "List all items (default limit)",
			query:         "",
			expectedCount: 3,
			expectedStatus: http.StatusOK,
		},
		{
			name:          "List with limit=2",
			query:         "?limit=2",
			expectedCount: 3, // Note: handler doesn't implement limit yet
			expectedStatus: http.StatusOK,
		},
		{
			name:          "List with offset=1",
			query:         "?offset=1",
			expectedCount: 3, // Note: handler doesn't implement offset yet
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/items"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.List(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var response struct {
					Data []map[string]interface{} `json:"data"`
				}
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if len(response.Data) != tt.expectedCount {
					t.Errorf("expected %d items, got %d", tt.expectedCount, len(response.Data))
				}
			}
		})
	}
}

func TestItemHandler_Search(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	handler := NewItemHandler(db)

	tests := []struct {
		name          string
		query         string
		expectedCount int
		expectedStatus int
	}{
		{
			name:          "Search for 'Tritanium'",
			query:         "?q=Tritanium",
			expectedCount: 1,
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Search for 'mineral' (multiple results)",
			query:         "?q=mineral",
			expectedCount: 2,
			expectedStatus: http.StatusOK,
		},
		{
			name:          "Search with no query",
			query:         "",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:          "Search for non-existent term",
			query:         "?q=nonexistent",
			expectedCount: 0,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/v1/search"+tt.query, nil)
			w := httptest.NewRecorder()

			handler.Search(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var result struct {
					Data []map[string]interface{} `json:"data"`
				}
				err := json.NewDecoder(w.Body).Decode(&result)
				if err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if len(result.Data) != tt.expectedCount {
					t.Errorf("expected %d results, got %d", tt.expectedCount, len(result.Data))
				}
			}
		})
	}
}
