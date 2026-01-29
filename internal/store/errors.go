package store

import "errors"

// ErrNotFound indicates the requested record does not exist.
var ErrNotFound = errors.New("task not found")
