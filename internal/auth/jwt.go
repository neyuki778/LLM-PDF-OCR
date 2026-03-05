package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Claims is the JWT payload used by auth service.
type Claims struct {
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

// JWTManager signs and verifies JWT tokens.
type JWTManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJWTManager(secret, issuer string, accessTTL, refreshTTL time.Duration) (*JWTManager, error) {
	if secret == "" {
		return nil, ErrEmptyJWTSecret
	}
	if issuer == "" {
		issuer = "llm-pdf-ocr"
	}
	if accessTTL <= 0 {
		accessTTL = 15 * time.Minute
	}
	if refreshTTL <= 0 {
		refreshTTL = 7 * 24 * time.Hour
	}
	return &JWTManager{
		secret:     []byte(secret),
		issuer:     issuer,
		accessTTL:  accessTTL,
		refreshTTL: refreshTTL,
	}, nil
}

func (m *JWTManager) IssueAccessToken(userID string, now time.Time) (string, time.Time, error) {
	return m.issue(userID, TokenTypeAccess, m.accessTTL, now)
}

func (m *JWTManager) IssueRefreshToken(userID string, now time.Time) (string, time.Time, error) {
	return m.issue(userID, TokenTypeRefresh, m.refreshTTL, now)
}

func (m *JWTManager) issue(userID, tokenType string, ttl time.Duration, now time.Time) (string, time.Time, error) {
	if userID == "" {
		return "", time.Time{}, ErrEmptyUserID
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	expiresAt := now.Add(ttl)

	claims := &Claims{
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign jwt: %w", err)
	}

	return signed, expiresAt, nil
}

func HashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (m *JWTManager) ParseAndValidate(rawToken, expectedType string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(rawToken, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidCredentials
	}
	if expectedType != "" && claims.TokenType != expectedType {
		return nil, ErrInvalidCredentials
	}
	return claims, nil
}
