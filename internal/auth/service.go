package auth

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const MinPasswordLength = 8

type Service struct {
	store *SQLiteStore
	jwt   *JWTManager
	nowFn func() time.Time
}

type ServiceConfig struct {
	JWTSecret  string
	JWTIssuer  string
	AccessTTL  time.Duration
	RefreshTTL time.Duration
}

type LoginResult struct {
	User               *User
	AccessToken        string
	AccessTokenExpires time.Time
	RefreshToken       string
	RefreshExpires     time.Time
}

func NewService(store *SQLiteStore, cfg ServiceConfig) (*Service, error) {
	if store == nil {
		return nil, ErrNilStore
	}
	jwtManager, err := NewJWTManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.AccessTTL, cfg.RefreshTTL)
	if err != nil {
		return nil, err
	}
	return &Service{
		store: store,
		jwt:   jwtManager,
		nowFn: func() time.Time { return time.Now().UTC() },
	}, nil
}

func (s *Service) Register(ctx context.Context, email, password string) (*User, error) {
	cleanEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := s.nowFn()
	user := &User{
		ID:           uuid.NewString(),
		Email:        cleanEmail,
		PasswordHash: hash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.store.CreateUser(ctx, user); err != nil {
		if errors.Is(err, ErrUserExists) {
			return nil, ErrUserExists
		}
		return nil, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*LoginResult, error) {
	cleanEmail, err := normalizeEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if password == "" {
		return nil, ErrInvalidCredentials
	}

	user, err := s.store.GetUserByEmail(ctx, cleanEmail)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := VerifyPassword(password, user.PasswordHash); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("verify password: %w", err)
	}

	now := s.nowFn()
	accessToken, accessExpires, err := s.jwt.IssueAccessToken(user.ID, now)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshExpires, err := s.jwt.IssueRefreshToken(user.ID, now)
	if err != nil {
		return nil, err
	}

	refresh := &RefreshToken{
		ID:        uuid.NewString(),
		UserID:    user.ID,
		TokenHash: HashToken(refreshToken),
		ExpiresAt: refreshExpires,
		CreatedAt: now,
	}
	if err := s.store.CreateRefreshToken(ctx, refresh); err != nil {
		return nil, err
	}

	return &LoginResult{
		User:               user,
		AccessToken:        accessToken,
		AccessTokenExpires: accessExpires,
		RefreshToken:       refreshToken,
		RefreshExpires:     refreshExpires,
	}, nil
}

func normalizeEmail(email string) (string, error) {
	clean := strings.TrimSpace(strings.ToLower(email))
	if clean == "" {
		return "", ErrEmptyEmail
	}
	addr, err := mail.ParseAddress(clean)
	if err != nil || addr.Address != clean {
		return "", ErrInvalidEmail
	}
	return clean, nil
}

func validatePassword(password string) error {
	if len(strings.TrimSpace(password)) < MinPasswordLength {
		return ErrPasswordTooShort
	}
	return nil
}
