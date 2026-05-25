package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

// APIKey represents an API key
type APIKey struct {
	ID        int64
	Key       string
	KeyHash   string
	Name      string
	RateLimit int // requests per minute
	CreatedAt time.Time
	ExpiresAt *time.Time
	Active    bool
}

// HashAPIKey returns a stable SHA-256 hash for storing API keys at rest.
func HashAPIKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// KeyPreview returns a non-secret display value for an API key.
func KeyPreview(key string) string {
	if len(key) <= 12 {
		return "********"
	}

	return key[:8] + "..." + key[len(key)-4:]
}

// Manager handles API key management
type Manager struct {
	db *sql.DB
}

// NewManager creates a new auth manager
func NewManager(db *sql.DB) *Manager {
	return &Manager{db: db}
}

// GenerateAPIKey generates a new random API key
func GenerateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "esk_" + hex.EncodeToString(bytes), nil
}

// CreateAPIKey creates a new API key
func (m *Manager) CreateAPIKey(ctx context.Context, name string, rateLimit int, expiresIn *time.Duration) (*APIKey, error) {
	if err := m.ensureKeyHashSchema(ctx); err != nil {
		return nil, err
	}

	key, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}
	keyHash := HashAPIKey(key)
	keyRef := keyReference(keyHash)

	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	}

	result, err := m.db.ExecContext(ctx, `
		INSERT INTO api_keys (key, key_hash, name, rate_limit, expires_at, active)
		VALUES (?, ?, ?, ?, ?, ?)
	`, keyRef, keyHash, name, rateLimit, expiresAt, true)
	if err != nil {
		return nil, err
	}

	id, _ := result.LastInsertId()

	log.Info().
		Int64("id", id).
		Str("name", name).
		Int("rate_limit", rateLimit).
		Msg("API key created")

	return &APIKey{
		ID:        id,
		Key:       key,
		KeyHash:   keyHash,
		Name:      name,
		RateLimit: rateLimit,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Active:    true,
	}, nil
}

// ValidateAPIKey validates an API key
func (m *Manager) ValidateAPIKey(ctx context.Context, key string) (*APIKey, error) {
	if err := m.ensureKeyHashSchema(ctx); err != nil {
		return nil, err
	}

	var apiKey APIKey
	var expiresAt sql.NullTime
	keyHash := HashAPIKey(key)

	err := m.db.QueryRowContext(ctx, `
		SELECT id, key, key_hash, name, rate_limit, created_at, expires_at, active
		FROM api_keys
		WHERE key_hash = ? AND active = 1
	`, keyHash).Scan(
		&apiKey.ID,
		&apiKey.Key,
		&apiKey.KeyHash,
		&apiKey.Name,
		&apiKey.RateLimit,
		&apiKey.CreatedAt,
		&expiresAt,
		&apiKey.Active,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInvalidAPIKey
	}
	if err != nil {
		return nil, err
	}
	apiKey.Key = keyHash
	apiKey.KeyHash = keyHash

	if expiresAt.Valid {
		apiKey.ExpiresAt = &expiresAt.Time
		if time.Now().After(*apiKey.ExpiresAt) {
			return nil, ErrExpiredAPIKey
		}
	}

	return &apiKey, nil
}

// RevokeAPIKey revokes an API key
func (m *Manager) RevokeAPIKey(ctx context.Context, key string) error {
	if err := m.ensureKeyHashSchema(ctx); err != nil {
		return err
	}

	keyHash := HashAPIKey(key)
	result, err := m.db.ExecContext(ctx, `
		UPDATE api_keys SET active = 0 WHERE key_hash = ? OR key = ?
	`, keyHash, key)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAPIKeyNotFound
	}

	log.Info().Str("key_hash", keyHash[:16]).Msg("API key revoked")
	return nil
}

// ListAPIKeys lists all API keys
func (m *Manager) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	if err := m.ensureKeyHashSchema(ctx); err != nil {
		return nil, err
	}

	rows, err := m.db.QueryContext(ctx, `
		SELECT id, key, key_hash, name, rate_limit, created_at, expires_at, active
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []APIKey
	for rows.Next() {
		var key APIKey
		var expiresAt sql.NullTime

		if err := rows.Scan(
			&key.ID,
			&key.Key,
			&key.KeyHash,
			&key.Name,
			&key.RateLimit,
			&key.CreatedAt,
			&expiresAt,
			&key.Active,
		); err != nil {
			return nil, err
		}

		if expiresAt.Valid {
			key.ExpiresAt = &expiresAt.Time
		}

		keys = append(keys, key)
	}

	return keys, nil
}

func (m *Manager) ensureKeyHashSchema(ctx context.Context) error {
	exists, err := m.columnExists(ctx, "api_keys", "key_hash")
	if err != nil {
		return err
	}
	if !exists {
		if _, err := m.db.ExecContext(ctx, `ALTER TABLE api_keys ADD COLUMN key_hash TEXT`); err != nil {
			exists, checkErr := m.columnExists(ctx, "api_keys", "key_hash")
			if checkErr != nil || !exists {
				return err
			}
		}
	}
	if _, err := m.db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_hash ON api_keys(key_hash) WHERE key_hash IS NOT NULL`); err != nil {
		return err
	}

	return m.migratePlaintextKeys(ctx)
}

func (m *Manager) columnExists(ctx context.Context, table, column string) (bool, error) {
	rows, err := m.db.QueryContext(ctx, "PRAGMA table_info("+table+")")
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, typ string
		var notNull int
		var defaultValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func (m *Manager) migratePlaintextKeys(ctx context.Context) error {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, key
		FROM api_keys
		WHERE key_hash IS NULL OR key_hash = ''
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type legacyKey struct {
		id  int64
		key string
	}
	var legacyKeys []legacyKey
	for rows.Next() {
		var item legacyKey
		if err := rows.Scan(&item.id, &item.key); err != nil {
			return err
		}
		legacyKeys = append(legacyKeys, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for _, item := range legacyKeys {
		keyHash := HashAPIKey(item.key)
		if _, err := m.db.ExecContext(ctx, `
			UPDATE api_keys
			SET key_hash = ?, key = ?
			WHERE id = ?
		`, keyHash, keyReference(keyHash), item.id); err != nil {
			return err
		}
	}

	_, err = m.db.ExecContext(ctx, `
		UPDATE api_keys
		SET key = 'sha256_' || key_hash
		WHERE key_hash IS NOT NULL
			AND key_hash <> ''
			AND key <> 'sha256_' || key_hash
	`)
	return err
}

func keyReference(keyHash string) string {
	return "sha256_" + keyHash
}

// Errors
var (
	ErrInvalidAPIKey  = errors.New("invalid API key")
	ErrExpiredAPIKey  = errors.New("expired API key")
	ErrAPIKeyNotFound = errors.New("API key not found")
)
