package errors

import "errors"

var (
	ErrInvalidURL   = errors.New("invalid URL")
	ErrCodeConflict = errors.New("short code already exists")
	ErrInvalidCode  = errors.New("invalid short code")
	ErrLinkNotFound = errors.New("link not found")
)
