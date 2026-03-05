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

// RotateRefreshToken revokes the old refresh token and inserts the new one atomically.
func (s *SQLiteStore) RotateRefreshToken(ctx context.Context, oldTokenHash string, newToken *RefreshToken, revokedAt time.Time) error {
	cleanOldHash := strings.TrimSpace(oldTokenHash)
	if cleanOldHash == "" {
		return ErrEmptyRefreshTokenHash
	}
	if newToken == nil {
		return errors.New("new refresh token is nil")
	}
	newToken.ID = strings.TrimSpace(newToken.ID)
	newToken.UserID = strings.TrimSpace(newToken.UserID)
	newToken.TokenHash = strings.TrimSpace(newToken.TokenHash)
	if newToken.ID == "" {
		return ErrEmptyRefreshTokenID
	}
	if newToken.UserID == "" {
		return ErrEmptyUserID
	}
	if newToken.TokenHash == "" {
		return ErrEmptyRefreshTokenHash
	}
	if newToken.ExpiresAt.IsZero() {
		return errors.New("new refresh token expires_at is zero")
	}
	if newToken.CreatedAt.IsZero() {
		newToken.CreatedAt = time.Now().UTC()
	}
	if revokedAt.IsZero() {
		revokedAt = time.Now().UTC()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx for refresh token rotation: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	updateResult, err := tx.ExecContext(
		ctx,
		`UPDATE refresh_tokens
		 SET revoked_at = ?
		 WHERE token_hash = ? AND revoked_at IS NULL;`,
		revokedAt.Unix(),
		cleanOldHash,
	)
	if err != nil {
		return fmt.Errorf("revoke old refresh token in tx: %w", err)
	}
	affected, err := updateResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("revoke old refresh token rows in tx: %w", err)
	}
	if affected == 0 {
		return ErrRefreshTokenNotFound
	}

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at, revoked_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?);`,
		newToken.ID,
		newToken.UserID,
		newToken.TokenHash,
		newToken.ExpiresAt.Unix(),
		nil,
		newToken.CreatedAt.Unix(),
	)
	if err != nil {
		if isUniqueConstraint(err) {
			return ErrRefreshTokenExists
		}
		return fmt.Errorf("insert new refresh token in tx: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit refresh token rotation: %w", err)
	}
	return nil
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
