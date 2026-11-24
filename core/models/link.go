package models

// Link связывает уникальный код с исходным URL.
type Link struct {
	Code string
	URL  string
}

// CodeLength is the number of base64url characters in a short code.
const CodeLength = 12

// MaxURLBytes limits the original URL before percent-encoding.
const MaxURLBytes = 8192
