package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// CreateRefreshToken inserts a refresh token record.
func (s *SQLiteStore) CreateRefreshToken(ctx context.Context, token *RefreshToken) error {
	if token == nil {
		return errors.New("refresh token is nil")
	}
	token.ID = strings.TrimSpace(token.ID)
	token.UserID = strings.TrimSpace(token.UserID)
	token.TokenHash = strings.TrimSpace(token.TokenHash)
	if token.ID == "" {
		return ErrEmptyRefreshTokenID
	}
	if token.UserID == "" {
		return ErrEmptyUserID
	}
	if token.TokenHash == "" {
		return ErrEmptyRefreshTokenHash
	}
	if token.ExpiresAt.IsZero() {
		return errors.New("refresh token expires_at is zero")
	}
	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now().UTC()
	}

	var revokedAt any
	if token.RevokedAt != nil {
		revokedAt = token.RevokedAt.Unix()
	}

	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?);`,
		token.ID,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt.Unix(),
		revokedAt,
		token.CreatedAt.Unix(),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrRefreshTokenExists
		}
		return fmt.Errorf("create refresh token: %w", err)
	}
	return nil
}

// GetRefreshTokenByHash returns a refresh token row by token hash.
func (s *SQLiteStore) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*RefreshToken, error) {
	cleanHash := strings.TrimSpace(tokenHash)
	if cleanHash == "" {
		return nil, ErrEmptyRefreshTokenHash
	}

	row := s.db.QueryRowContext(
		ctx,
		`SELECT id, user_id, token_hash, expires_at, revoked_at, created_at
		 FROM refresh_tokens
		 WHERE token_hash = ?
		 LIMIT 1;`,
		cleanHash,
	)

	token, err := scanRefreshToken(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRefreshTokenNotFound
		}
		return nil, fmt.Errorf("get refresh token by hash: %w", err)
	}
	return token, nil
}

// RevokeRefreshTokenByHash marks a refresh token as revoked.
func (s *SQLiteStore) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string, revokedAt time.Time) error {
	cleanHash := strings.TrimSpace(tokenHash)
	if cleanHash == "" {
		return ErrEmptyRefreshTokenHash
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}

	result, err := s.db.ExecContext(
		ctx,
		`UPDATE refresh_tokens
		 SET revoked_at = ?
		 WHERE token_hash = ? AND revoked_at IS NULL;`,
		revokedAt.Unix(),
		cleanHash,
	)
	if err != nil {
		return fmt.Errorf("revoke refresh token: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke refresh token rows: %w", err)
	}
	if rows == 0 {
		return ErrRefreshTokenNotFound
	}
	return nil
}

// DeleteExpiredRefreshTokens removes expired rows and returns affected count.
func (s *SQLiteStore) DeleteExpiredRefreshTokens(ctx context.Context, now time.Time) (int64, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}

	result, err := s.db.ExecContext(
		ctx,
		`DELETE FROM refresh_tokens WHERE expires_at < ?;`,
		now.Unix(),
	)
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("delete expired refresh tokens rows: %w", err)
	}
	return rows, nil
}

func scanRefreshToken(scanner scanTarget) (*RefreshToken, error) {
	var (
		token     RefreshToken
		expiresAt int64
		createdAt int64
		revokedAt sql.NullInt64
	)

	if err := scanner.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&expiresAt,
		&revokedAt,
		&createdAt,
	); err != nil {
		return nil, err
	}

	token.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	token.CreatedAt = time.Unix(createdAt, 0).UTC()
	if revokedAt.Valid {
		t := time.Unix(revokedAt.Int64, 0).UTC()
		token.RevokedAt = &t
	}

	return &token, nil
}
