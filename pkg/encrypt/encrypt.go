package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
)

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// ErrCiphertextTooShort is returned by [Decrypt] when the data is shorter than
// the nonce which prefixes every ciphertext.
var ErrCiphertextTooShort = errors.New("ciphertext too short")

// EncryptString encrypts data with AES-GCM using a key derived from passphrase.
func EncryptString(data string, passphrase string) (string, error) {
	b, err := Encrypt([]byte(data), passphrase)
	return string(b), err
}

// DecryptString decrypts data produced by [EncryptString].
func DecryptString(data string, passphrase string) (string, error) {
	b, err := Decrypt([]byte(data), passphrase)
	return string(b), err
}

// Encrypt encrypts data with AES-GCM using a key derived from passphrase.
func Encrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(createHash(passphrase)))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypt decrypts data produced by [Encrypt]. It returns an error (never
// panics) when the passphrase is wrong or the data is truncated or corrupt.
func Decrypt(data []byte, passphrase string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(createHash(passphrase)))
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, ErrCiphertextTooShort
	}
	return gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
}

// EncryptFile encrypts data with [Encrypt] and writes it to filename (created
// or truncated).
func EncryptFile(filename string, data []byte, passphrase string) error {
	enc, err := Encrypt(data, passphrase)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, enc, 0o600)
}

// DecryptFile reads filename and decrypts it with [Decrypt].
func DecryptFile(filename string, passphrase string) ([]byte, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return Decrypt(data, passphrase)
}
