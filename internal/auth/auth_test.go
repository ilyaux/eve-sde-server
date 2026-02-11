package auth

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	// Create api_keys table
	_, err = db.Exec(`
		CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			rate_limit INTEGER NOT NULL DEFAULT 1000,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			expires_at TIMESTAMP,
			active BOOLEAN NOT NULL DEFAULT 1
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	return db
}

func TestGenerateAPIKey(t *testing.T) {
	key, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if len(key) == 0 {
		t.Error("generated key is empty")
	}

	if key[:4] != "esk_" {
		t.Errorf("key should start with 'esk_', got %s", key[:4])
	}

	// Generate another key and ensure it's different
	key2, err := GenerateAPIKey()
	if err != nil {
		t.Fatalf("GenerateAPIKey() error = %v", err)
	}

	if key == key2 {
		t.Error("generated keys should be unique")
	}
}

func TestCreateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	apiKey, err := manager.CreateAPIKey(ctx, "test-key", 1000, nil)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	if apiKey.Name != "test-key" {
		t.Errorf("expected name 'test-key', got %s", apiKey.Name)
	}

	if apiKey.RateLimit != 1000 {
		t.Errorf("expected rate limit 1000, got %d", apiKey.RateLimit)
	}

	if !apiKey.Active {
		t.Error("expected key to be active")
	}

	if apiKey.ExpiresAt != nil {
		t.Error("expected no expiration")
	}
}

func TestCreateAPIKey_WithExpiration(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	expiration := 24 * time.Hour
	apiKey, err := manager.CreateAPIKey(ctx, "test-key", 1000, &expiration)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	if apiKey.ExpiresAt == nil {
		t.Fatal("expected expiration to be set")
	}

	// Check expiration is approximately 24 hours from now
	diff := time.Until(*apiKey.ExpiresAt)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expected expiration in ~24 hours, got %v", diff)
	}
}

func TestValidateAPIKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	// Create a key
	created, err := manager.CreateAPIKey(ctx, "test-key", 1000, nil)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	// Validate it
	validated, err := manager.ValidateAPIKey(ctx, created.Key)
	if err != nil {
		t.Fatalf("ValidateAPIKey() error = %v", err)
	}

	if validated.ID != created.ID {
		t.Errorf("expected ID %d, got %d", created.ID, validated.ID)
	}

	if validated.Name != created.Name {
		t.Errorf("expected name %s, got %s", created.Name, validated.Name)
	}
}

func TestValidateAPIKey_Invalid(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	_, err := manager.ValidateAPIKey(ctx, "invalid-key")
	if err != ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestValidateAPIKey_Expired(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	// Create a key that expires immediately
	expiration := 1 * time.Nanosecond
	created, err := manager.CreateAPIKey(ctx, "test-key", 1000, &expiration)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	// Wait for it to expire
	time.Sleep(10 * time.Millisecond)

	_, err = manager.ValidateAPIKey(ctx, created.Key)
	if err != ErrExpiredAPIKey {
		t.Errorf("expected ErrExpiredAPIKey, got %v", err)
	}
}

func TestRevokeAPIKey(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	// Create and revoke a key
	created, err := manager.CreateAPIKey(ctx, "test-key", 1000, nil)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	err = manager.RevokeAPIKey(ctx, created.Key)
	if err != nil {
		t.Fatalf("RevokeAPIKey() error = %v", err)
	}

	// Validation should fail
	_, err = manager.ValidateAPIKey(ctx, created.Key)
	if err != ErrInvalidAPIKey {
		t.Errorf("expected ErrInvalidAPIKey after revocation, got %v", err)
	}
}

func TestListAPIKeys(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	manager := NewManager(db)
	ctx := context.Background()

	// Create multiple keys
	_, err := manager.CreateAPIKey(ctx, "key1", 1000, nil)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	_, err = manager.CreateAPIKey(ctx, "key2", 2000, nil)
	if err != nil {
		t.Fatalf("CreateAPIKey() error = %v", err)
	}

	keys, err := manager.ListAPIKeys(ctx)
	if err != nil {
		t.Fatalf("ListAPIKeys() error = %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}
