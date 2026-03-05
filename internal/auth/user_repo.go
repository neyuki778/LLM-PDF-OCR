package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CreateUser inserts a user record.
func (s *SQLiteStore) CreateUser(ctx context.Context, user *User) error {
	if user == nil {
		return errors.New("user is nil")
	}
	user.ID = strings.TrimSpace(user.ID)
	user.Email = strings.TrimSpace(strings.ToLower(user.Email))
	user.PasswordHash = strings.TrimSpace(user.PasswordHash)
	if user.ID == "" {
		return ErrEmptyUserID
	}
	if user.Email == "" {
		return ErrEmptyEmail
	}
	if user.PasswordHash == "" {
		return ErrEmptyPasswordHash
	}

	now := time.Now().UTC()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO users (id, email, password_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?);`,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.CreatedAt.Unix(),
		user.UpdatedAt.Unix(),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrUserExists
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

// GetUserByEmail returns a user by email.
func (s *SQLiteStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	cleanEmail := strings.TrimSpace(strings.ToLower(email))
	if cleanEmail == "" {
		return nil, ErrEmptyEmail
	}

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM users
		 WHERE email = ?
		 LIMIT 1;`,
		cleanEmail,
	)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	return user, nil
}

// GetUserByID returns a user by id.
func (s *SQLiteStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	cleanID := strings.TrimSpace(id)
	if cleanID == "" {
		return nil, ErrEmptyUserID
	}

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, email, password_hash, created_at, updated_at
		 FROM users
		 WHERE id = ?
		 LIMIT 1;`,
		cleanID,
	)

	user, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return user, nil
}

type scanTarget interface {
	Scan(dest ...any) error
}

func scanUser(scanner scanTarget) (*User, error) {
	var (
		user      User
		createdAt int64
		updatedAt int64
	)

	if err := scanner.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&createdAt,
		&updatedAt,
	); err != nil {
		return nil, err
	}

	user.CreatedAt = time.Unix(createdAt, 0).UTC()
	user.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return &user, nil
}
