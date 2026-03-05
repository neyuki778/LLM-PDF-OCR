package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "auth.db")
	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("new sqlite store: %v", err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestUserRepo_CreateAndGet(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	user := &User{
		ID:           "user_1",
		Email:        "USER@example.com",
		PasswordHash: "bcrypt-hash",
	}
	if err := store.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	gotByEmail, err := store.GetUserByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get user by email: %v", err)
	}
	if gotByEmail.ID != "user_1" {
		t.Fatalf("unexpected id: %s", gotByEmail.ID)
	}
	if gotByEmail.Email != "user@example.com" {
		t.Fatalf("email should be normalized, got: %s", gotByEmail.Email)
	}

	gotByID, err := store.GetUserByID(ctx, "user_1")
	if err != nil {
		t.Fatalf("get user by id: %v", err)
	}
	if gotByID.Email != "user@example.com" {
		t.Fatalf("unexpected email: %s", gotByID.Email)
	}
}

func TestUserRepo_DuplicateEmail(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.CreateUser(ctx, &User{ID: "u1", Email: "a@example.com", PasswordHash: "h1"}); err != nil {
		t.Fatalf("create first user: %v", err)
	}

	err := store.CreateUser(ctx, &User{ID: "u2", Email: "a@example.com", PasswordHash: "h2"})
	if !errors.Is(err, ErrUserExists) {
		t.Fatalf("expected ErrUserExists, got: %v", err)
	}
}

func TestRefreshTokenRepo_CRUDLikeFlow(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	if err := store.CreateUser(ctx, &User{ID: "u1", Email: "a@example.com", PasswordHash: "h1"}); err != nil {
		t.Fatalf("create user: %v", err)
	}

	expiresAt := time.Now().UTC().Add(1 * time.Hour)
	token := &RefreshToken{
		ID:        "rt_1",
		UserID:    "u1",
		TokenHash: "hash_1",
		ExpiresAt: expiresAt,
	}
	if err := store.CreateRefreshToken(ctx, token); err != nil {
		t.Fatalf("create refresh token: %v", err)
	}

	got, err := store.GetRefreshTokenByHash(ctx, "hash_1")
	if err != nil {
		t.Fatalf("get refresh token: %v", err)
	}
	if got.UserID != "u1" {
		t.Fatalf("unexpected user id: %s", got.UserID)
	}
	if got.RevokedAt != nil {
		t.Fatalf("expected not revoked")
	}

	if err := store.RevokeRefreshTokenByHash(ctx, "hash_1", time.Time{}); err != nil {
		t.Fatalf("revoke refresh token: %v", err)
	}

	got, err = store.GetRefreshTokenByHash(ctx, "hash_1")
	if err != nil {
		t.Fatalf("get refresh token after revoke: %v", err)
	}
	if got.RevokedAt == nil {
		t.Fatalf("expected revoked token")
	}

	if _, err := store.DeleteExpiredRefreshTokens(ctx, time.Now().UTC()); err != nil {
		t.Fatalf("delete expired should not fail: %v", err)
	}

	expired := &RefreshToken{
		ID:        "rt_2",
		UserID:    "u1",
		TokenHash: "hash_2",
		ExpiresAt: time.Now().UTC().Add(-1 * time.Hour),
	}
	if err := store.CreateRefreshToken(ctx, expired); err != nil {
		t.Fatalf("create expired refresh token: %v", err)
	}

	deleted, err := store.DeleteExpiredRefreshTokens(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("delete expired refresh tokens: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("expected 1 deleted row, got %d", deleted)
	}
}
