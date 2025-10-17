package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
)

type CryptoService struct {
	key []byte
}

// NewCryptoService creates a new crypto service with the given encryption key
func NewCryptoService(encryptionKey string) *CryptoService {
	// Derive a 32-byte key from the provided key using SHA-256
	hash := sha256.Sum256([]byte(encryptionKey))
	return &CryptoService{
		key: hash[:],
	}
}

// Encrypt encrypts plaintext using AES-256-GCM
func (s *CryptoService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Create a nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt the data
	ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
	
	// Encode to base64 for storage
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext encrypted with Encrypt
func (s *CryptoService) Decrypt(ciphertext string) (string, error) {
	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := aesGCM.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	// Extract the nonce from the beginning of the ciphertext
	nonce, ciphertext := data[:nonceSize], string(data[nonceSize:])

	// Decrypt the data
	plaintext, err := aesGCM.Open(nil, nonce, []byte(ciphertext), nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}