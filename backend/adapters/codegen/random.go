package codegen

import (
	"crypto/rand"
	"encoding/base64"
)

type Random struct{}

func (Random) Generate() (string, error) {
	var value [9]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value[:]), nil
}
