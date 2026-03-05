package auth

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultBusyTimeoutMS = 5000

var ErrEmptyDBPath = errors.New("sqlite path is empty")

type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore opens (or creates) sqlite db file and runs auth migrations.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	cleanPath := strings.TrimSpace(dbPath)
	if cleanPath == "" {
		return nil, ErrEmptyDBPath
	}

	if err := ensureSQLiteFile(cleanPath); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", cleanPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite works best with single writer connection in this service.
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

	if err := migrateAuthSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// Close releases sqlite resources.
func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// DB exposes sql.DB for repository-layer methods in the next step.
func (s *SQLiteStore) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.db
}

func migrateAuthSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at INTEGER NOT NULL,
			updated_at INTEGER NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);`,
		`CREATE TABLE IF NOT EXISTS refresh_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at INTEGER NOT NULL,
			revoked_at INTEGER,
			created_at INTEGER NOT NULL,
			FOREIGN KEY(user_id) REFERENCES users(id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("migrate auth schema: %w", err)
		}
	}
	return nil
}

func ensureSQLiteFile(path string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create db dir: %w", err)
		}
	}

	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("create db file: %w", err)
	}
	return f.Close()
}
