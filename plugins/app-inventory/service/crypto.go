package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

const secretKeyEnv = "OPSHUB_APP_INVENTORY_SECRET_KEY"

// SecretCipher encrypts credentials with AES-256-GCM. The master key is never persisted in MySQL.
type SecretCipher struct {
	key        []byte
	keyVersion string
}

func NewSecretCipher(rawKey string) (*SecretCipher, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return nil, fmt.Errorf("%s is not configured", secretKeyEnv)
	}

	// Accept a base64 encoded 32-byte key, while hashing other strong values for easier deployment.
	key := []byte(rawKey)
	if decoded, err := base64.RawStdEncoding.DecodeString(rawKey); err == nil && len(decoded) == 32 {
		key = decoded
	}
	sum := sha256.Sum256(key)
	version := fmt.Sprintf("sha256:%x", sum[:6])
	return &SecretCipher{key: sum[:], keyVersion: version}, nil
}

func (c *SecretCipher) KeyVersion() string {
	if c == nil {
		return ""
	}
	return c.keyVersion
}

func (c *SecretCipher) Encrypt(plaintext []byte) (string, error) {
	return c.EncryptWithAAD(plaintext, []byte("opshub:app-inventory:v1"))
}

func (c *SecretCipher) EncryptWithAAD(plaintext, aad []byte) (string, error) {
	if c == nil || len(c.key) != 32 {
		return "", errors.New("secret cipher is not initialized")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	sealed := aead.Seal(nil, nonce, plaintext, aad)
	encoded := append(nonce, sealed...)
	return base64.RawStdEncoding.EncodeToString(encoded), nil
}

func (c *SecretCipher) Decrypt(ciphertext string) ([]byte, error) {
	return c.DecryptWithAAD(ciphertext, []byte("opshub:app-inventory:v1"))
}

func (c *SecretCipher) DecryptWithAAD(ciphertext string, aad []byte) ([]byte, error) {
	if c == nil || len(c.key) != 32 {
		return nil, errors.New("secret cipher is not initialized")
	}
	data, err := base64.RawStdEncoding.DecodeString(strings.TrimSpace(ciphertext))
	if err != nil {
		return nil, fmt.Errorf("decode credential ciphertext: %w", err)
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(data) < aead.NonceSize() {
		return nil, errors.New("credential ciphertext is truncated")
	}
	return aead.Open(nil, data[:aead.NonceSize()], data[aead.NonceSize():], aad)
}
