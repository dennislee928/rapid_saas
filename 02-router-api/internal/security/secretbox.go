package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const encryptedPrefix = "enc:v1:"

func EncryptSecret(keyMaterial, plaintext string) (string, error) {
	if strings.TrimSpace(keyMaterial) == "" {
		return "", errors.New("key material is required")
	}
	block, err := aes.NewCipher(deriveKey(keyMaterial))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return encryptedPrefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

func DecryptSecret(keyMaterial, ciphertext string) (string, error) {
	if !strings.HasPrefix(ciphertext, encryptedPrefix) {
		return "", errors.New("unsupported ciphertext format")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, encryptedPrefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(deriveKey(keyMaterial))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, sealed := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func RedactSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 8 {
		return "redacted"
	}
	return value[:4] + "...redacted..." + value[len(value)-4:]
}

func deriveKey(keyMaterial string) []byte {
	sum := sha256.Sum256([]byte(keyMaterial))
	return sum[:]
}
