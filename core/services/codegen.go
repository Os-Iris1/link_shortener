package services

import (
	"crypto/rand"
	"encoding/base64"

	"shorter/core/models"
)

// codeEntropyBytes yields CodeLength characters in unpadded base64 (6 bits each).
const (
	base64BitsPerCharacter = 6
	bitsPerByte            = 8
	codeEntropyBytes       = models.CodeLength * base64BitsPerCharacter / bitsPerByte
)

type Random struct{}

func (Random) Generate() (string, error) {
	var value [codeEntropyBytes]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}
