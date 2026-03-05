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
