package credential

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

const MinAdminTokenLength = 20

func Generate() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func Hash(token string) [sha256.Size]byte {
	return sha256.Sum256([]byte(token))
}
