package auth

import "errors"

var (
	ErrUserExists            = errors.New("user already exists")
	ErrUserNotFound          = errors.New("user not found")
	ErrRefreshTokenExists    = errors.New("refresh token already exists")
	ErrRefreshTokenNotFound  = errors.New("refresh token not found")
	ErrEmptyUserID           = errors.New("user id is empty")
	ErrEmptyEmail            = errors.New("email is empty")
	ErrEmptyPasswordHash     = errors.New("password hash is empty")
	ErrEmptyRefreshTokenID   = errors.New("refresh token id is empty")
	ErrEmptyRefreshTokenHash = errors.New("refresh token hash is empty")
)
