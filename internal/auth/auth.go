package auth

import (
	"context"
	"crypto/rand"
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
	Name      string
	RateLimit int       // requests per minute
	CreatedAt time.Time
	ExpiresAt *time.Time
	Active    bool
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
	key, err := GenerateAPIKey()
	if err != nil {
		return nil, err
	}

	var expiresAt *time.Time
	if expiresIn != nil {
		t := time.Now().Add(*expiresIn)
		expiresAt = &t
	}

	result, err := m.db.ExecContext(ctx, `
		INSERT INTO api_keys (key, name, rate_limit, expires_at, active)
		VALUES (?, ?, ?, ?, ?)
	`, key, name, rateLimit, expiresAt, true)
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
		Name:      name,
		RateLimit: rateLimit,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		Active:    true,
	}, nil
}

// ValidateAPIKey validates an API key
func (m *Manager) ValidateAPIKey(ctx context.Context, key string) (*APIKey, error) {
	var apiKey APIKey
	var expiresAt sql.NullTime

	err := m.db.QueryRowContext(ctx, `
		SELECT id, key, name, rate_limit, created_at, expires_at, active
		FROM api_keys
		WHERE key = ? AND active = 1
	`, key).Scan(
		&apiKey.ID,
		&apiKey.Key,
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
	result, err := m.db.ExecContext(ctx, `
		UPDATE api_keys SET active = 0 WHERE key = ?
	`, key)
	if err != nil {
		return err
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrAPIKeyNotFound
	}

	log.Info().Str("key", key[:16]+"...").Msg("API key revoked")
	return nil
}

// ListAPIKeys lists all API keys
func (m *Manager) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := m.db.QueryContext(ctx, `
		SELECT id, key, name, rate_limit, created_at, expires_at, active
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

// Errors
var (
	ErrInvalidAPIKey  = errors.New("invalid API key")
	ErrExpiredAPIKey  = errors.New("expired API key")
	ErrAPIKeyNotFound = errors.New("API key not found")
)
