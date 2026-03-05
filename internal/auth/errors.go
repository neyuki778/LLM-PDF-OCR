package auth

import "errors"

var (
	ErrUserExists            = errors.New("user already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrRefreshTokenExists    = errors.New("refresh token already exists")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrInvalidRefreshToken   = errors.New("invalid refresh token")
	ErrInvalidEmail          = errors.New("invalid email")
	ErrPasswordTooShort      = errors.New("password too short")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrEmptyJWTSecret        = errors.New("jwt secret is empty")
	ErrNilStore              = errors.New("auth store is nil")
	ErrEmptyUserID           = errors.New("user id is empty")
	ErrEmptyEmail            = errors.New("email is empty")
	ErrEmptyPasswordHash     = errors.New("password hash is empty")
	ErrEmptyRefreshTokenID   = errors.New("refresh token id is empty")
	ErrEmptyRefreshTokenHash = errors.New("refresh token hash is empty")
)
