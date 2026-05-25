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

func setupTaxonomyTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	_, err = db.Exec(`
		CREATE TABLE categories (
			category_id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT 1
		);
		CREATE TABLE groups (
			group_id INTEGER PRIMARY KEY,
			category_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			published BOOLEAN NOT NULL DEFAULT 1
		);
		INSERT INTO categories (category_id, name, published) VALUES
			(4, 'Material', 1),
			(6, 'Ship', 1);
		INSERT INTO groups (group_id, category_id, name, published) VALUES
			(18, 4, 'Mineral', 1),
			(25, 6, 'Frigate', 1),
			(26, 6, 'Cruiser', 1);
	`)
	if err != nil {
		t.Fatalf("create fixture schema: %v", err)
	}

	return db
}

func TestTaxonomyHandler_ListCategories(t *testing.T) {
	db := setupTaxonomyTestDB(t)
	defer db.Close()

	handler := NewTaxonomyHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories?limit=1", nil)
	w := httptest.NewRecorder()

	handler.ListCategories(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Data []Category `json:"data"`
		Meta struct {
			Count int `json:"count"`
			Total int `json:"total"`
			Limit int `json:"limit"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 1 || response.Meta.Total != 2 || response.Meta.Limit != 1 {
		t.Fatalf("unexpected response: %+v", response)
	}
	if response.Data[0].CategoryID != 4 || response.Data[0].Name != "Material" {
		t.Fatalf("unexpected category: %+v", response.Data[0])
	}
}

func TestTaxonomyHandler_GetCategory(t *testing.T) {
	db := setupTaxonomyTestDB(t)
	defer db.Close()

	handler := NewTaxonomyHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/categories/6", nil)
	w := httptest.NewRecorder()
	req = withURLParam(req, "id", "6")

	handler.GetCategory(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var category Category
	if err := json.NewDecoder(w.Body).Decode(&category); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if category.CategoryID != 6 || category.Name != "Ship" {
		t.Fatalf("unexpected category: %+v", category)
	}
}

func TestTaxonomyHandler_ListGroupsByCategory(t *testing.T) {
	db := setupTaxonomyTestDB(t)
	defer db.Close()

	handler := NewTaxonomyHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups?category_id=6", nil)
	w := httptest.NewRecorder()

	handler.ListGroups(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var response struct {
		Data []Group `json:"data"`
		Meta struct {
			Count int `json:"count"`
			Total int `json:"total"`
		} `json:"meta"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data) != 2 || response.Meta.Total != 2 {
		t.Fatalf("unexpected response: %+v", response)
	}
	for _, group := range response.Data {
		if group.CategoryID != 6 {
			t.Fatalf("unexpected group category: %+v", group)
		}
	}
}

func TestTaxonomyHandler_GetGroup(t *testing.T) {
	db := setupTaxonomyTestDB(t)
	defer db.Close()

	handler := NewTaxonomyHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/18", nil)
	w := httptest.NewRecorder()
	req = withURLParam(req, "id", "18")

	handler.GetGroup(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	var group Group
	if err := json.NewDecoder(w.Body).Decode(&group); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if group.GroupID != 18 || group.Name != "Mineral" || group.CategoryID != 4 {
		t.Fatalf("unexpected group: %+v", group)
	}
}

func withURLParam(req *http.Request, key, value string) *http.Request {
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(key, value)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rctx)
	return req.WithContext(ctx)
}
