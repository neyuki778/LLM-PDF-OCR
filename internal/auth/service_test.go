package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestService(t *testing.T) (*SQLiteStore, *Service) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "auth_service.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})

	svc, err := NewService(store, ServiceConfig{
		JWTSecret:  "test-secret-1234567890",
		JWTIssuer:  "test-suite",
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("new auth service: %v", err)
	}
	return store, svc
}

func TestService_RegisterAndLogin(t *testing.T) {
	store, svc := newTestService(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, "USER@EXAMPLE.COM", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if user.Email != "user@example.com" {
		t.Fatalf("expected normalized email, got %s", user.Email)
	}
	if user.PasswordHash == "password123" {
		t.Fatalf("password hash should not equal raw password")
	}

	login, err := svc.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if login.AccessToken == "" || login.RefreshToken == "" {
		t.Fatalf("tokens should not be empty")
	}

	claims, err := svc.jwt.ParseAndValidate(login.AccessToken, TokenTypeAccess)
	if err != nil {
		t.Fatalf("parse access token: %v", err)
	}
	if claims.Subject != user.ID {
		t.Fatalf("unexpected sub: %s", claims.Subject)
	}

	stored, err := store.GetRefreshTokenByHash(ctx, HashToken(login.RefreshToken))
	if err != nil {
		t.Fatalf("refresh token should be persisted: %v", err)
	}
	if stored.UserID != user.ID {
		t.Fatalf("unexpected refresh token user id: %s", stored.UserID)
	}
}

func TestService_RegisterValidation(t *testing.T) {
	_, svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "bad-email", "password123")
	if !errors.Is(err, ErrInvalidEmail) {
		t.Fatalf("expected ErrInvalidEmail, got %v", err)
	}

	_, err = svc.Register(ctx, "user@example.com", "short")
	if !errors.Is(err, ErrPasswordTooShort) {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestService_LoginInvalidCredentials(t *testing.T) {
	_, svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	_, err = svc.Login(ctx, "user@example.com", "wrong-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}

	_, err = svc.Login(ctx, "missing@example.com", "password123")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestService_RefreshRotation(t *testing.T) {
	store, svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	login, err := svc.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	oldHash := HashToken(login.RefreshToken)
	refreshed, err := svc.Refresh(ctx, login.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatalf("refreshed tokens should not be empty")
	}
	if refreshed.RefreshToken == login.RefreshToken {
		t.Fatalf("refresh token should be rotated")
	}

	oldRecord, err := store.GetRefreshTokenByHash(ctx, oldHash)
	if err != nil {
		t.Fatalf("get old refresh token: %v", err)
	}
	if oldRecord.RevokedAt == nil {
		t.Fatalf("old refresh token should be revoked after rotation")
	}

	if _, err := store.GetRefreshTokenByHash(ctx, HashToken(refreshed.RefreshToken)); err != nil {
		t.Fatalf("new refresh token should be persisted: %v", err)
	}

	_, err = svc.Refresh(ctx, login.RefreshToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("reusing rotated refresh token should fail, got %v", err)
	}
}

func TestService_RefreshExpiredRecord(t *testing.T) {
	store, svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	login, err := svc.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	_, err = store.DB().ExecContext(
		ctx,
		`UPDATE refresh_tokens SET expires_at = ? WHERE token_hash = ?;`,
		time.Now().UTC().Add(-1*time.Hour).Unix(),
		HashToken(login.RefreshToken),
	)
	if err != nil {
		t.Fatalf("force expire refresh token in db: %v", err)
	}

	_, err = svc.Refresh(ctx, login.RefreshToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected ErrInvalidRefreshToken for expired record, got %v", err)
	}
}

func TestService_LogoutRevokesRefreshToken(t *testing.T) {
	store, svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	login, err := svc.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	if err := svc.Logout(ctx, login.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}

	record, err := store.GetRefreshTokenByHash(ctx, HashToken(login.RefreshToken))
	if err != nil {
		t.Fatalf("get refresh token after logout: %v", err)
	}
	if record.RevokedAt == nil {
		t.Fatalf("refresh token should be revoked")
	}

	_, err = svc.Refresh(ctx, login.RefreshToken)
	if !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("revoked refresh token should not refresh, got %v", err)
	}
}

func TestService_LogoutIsIdempotent(t *testing.T) {
	_, svc := newTestService(t)
	ctx := context.Background()

	if err := svc.Logout(ctx, ""); err != nil {
		t.Fatalf("empty token logout should be noop, got %v", err)
	}
	if err := svc.Logout(ctx, "not-a-real-token"); err != nil {
		t.Fatalf("missing token logout should be noop, got %v", err)
	}
	if err := svc.Logout(ctx, "not-a-real-token"); err != nil {
		t.Fatalf("repeated logout should be noop, got %v", err)
	}
}
