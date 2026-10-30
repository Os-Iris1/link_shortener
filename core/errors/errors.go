package errors

import "errors"

var (
	ErrInvalidURL   = errors.New("invalid URL")
	ErrCodeConflict = errors.New("short code already exists")
)
