package keystore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultBusyTimeoutMS = 5000
)

var (
	ErrKeyExists = errors.New("key already exists")
	ErrNotFound  = errors.New("key not found")
	ErrNoKeys    = errors.New("no keys available")
	ErrEmptyKey  = errors.New("key is empty")
	ErrEmptyPath = errors.New("db path is empty")
)

// Store manages Gemini API keys in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the sqlite database at dbPath and runs migrations.
func NewStore(dbPath string) (*Store, error) {
	cleanPath := strings.TrimSpace(dbPath)
	if cleanPath == "" {
		return nil, ErrEmptyPath
	}

	if err := ensureFile(cleanPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", cleanPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite works best with a single writer connection.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set journal_mode: %w", err)
	}
	if _, err := db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d;", defaultBusyTimeoutMS)); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close releases database resources.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// AddKey inserts a new key (enabled by default).
func (s *Store) AddKey(ctx context.Context, key, note string) (*Key, error) {
	cleanKey := strings.TrimSpace(key)
	if cleanKey == "" {
		return nil, ErrEmptyKey
	}

	now := time.Now().UTC().Unix()
	result, err := s.db.ExecContext(
		ctx,
		`INSERT INTO gemini_keys (key, note, enabled, created_at, updated_at)
		 VALUES (?, ?, 1, ?, ?);`,
		cleanKey,
		note,
		now,
		now,
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return nil, ErrKeyExists
		}
		return nil, fmt.Errorf("insert key: %w", err)
	}

	id, _ := result.LastInsertId()
	return &Key{
		ID:        id,
		Key:       cleanKey,
		Note:      note,
		Enabled:   true,
		CreatedAt: time.Unix(now, 0).UTC(),
		UpdatedAt: time.Unix(now, 0).UTC(),
	}, nil
}

// DeleteKey removes a key by id.
func (s *Store) DeleteKey(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM gemini_keys WHERE id = ?;`, id)
	if err != nil {
		return fmt.Errorf("delete key: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete key rows: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// GetKey returns a key by id.
func (s *Store) GetKey(ctx context.Context, id int64) (*Key, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, key, note, enabled, created_at, updated_at FROM gemini_keys WHERE id = ?;`, id)
	key, err := scanKey(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get key: %w", err)
	}
	return &key, nil
}

// ListKeys returns all keys.
func (s *Store) ListKeys(ctx context.Context) ([]Key, error) {
	return s.listKeys(ctx, false)
}

// ListEnabledKeys returns only enabled keys. If none, returns ErrNoKeys.
func (s *Store) ListEnabledKeys(ctx context.Context) ([]Key, error) {
	keys, err := s.listKeys(ctx, true)
	if err != nil {
		return nil, err
	}
	if len(keys) == 0 {
		return nil, ErrNoKeys
	}
	return keys, nil
}

// CountEnabledKeys returns the number of enabled keys.
func (s *Store) CountEnabledKeys(ctx context.Context) (int, error) {
	row := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM gemini_keys WHERE enabled = 1;`)
	var count int
	if err := row.Scan(&count); err != nil {
		return 0, fmt.Errorf("count keys: %w", err)
	}
	return count, nil
}

func (s *Store) listKeys(ctx context.Context, enabledOnly bool) ([]Key, error) {
	query := `SELECT id, key, note, enabled, created_at, updated_at FROM gemini_keys`
	if enabledOnly {
		query += ` WHERE enabled = 1`
	}
	query += ` ORDER BY id ASC;`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list keys: %w", err)
	}
	defer rows.Close()

	keys := make([]Key, 0, 8)
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, fmt.Errorf("scan key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list keys rows: %w", err)
	}
	return keys, nil
}

func migrate(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS gemini_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT NOT NULL UNIQUE,
			note TEXT,
			enabled INTEGER NOT NULL DEFAULT 1,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_gemini_keys_enabled ON gemini_keys(enabled);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanKey(scanner rowScanner) (Key, error) {
	var (
		key       Key
		enabled   int
		createdAt int64
		updatedAt int64
	)

	if err := scanner.Scan(&key.ID, &key.Key, &key.Note, &enabled, &createdAt, &updatedAt); err != nil {
		return Key{}, err
	}
	key.Enabled = enabled == 1
	key.CreatedAt = time.Unix(createdAt, 0).UTC()
	key.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return key, nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

func ensureFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create db dir: %w", err)
		}
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("create db file: %w", err)
	}
	return file.Close()
}
