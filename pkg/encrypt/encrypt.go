package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"os"
)

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

// ErrCiphertextTooShort is returned by [DecryptE] when the data is shorter than
// the nonce which prefixes every ciphertext.
var ErrCiphertextTooShort = errors.New("ciphertext too short")

// EncryptStringE encrypts data with AES-GCM using a key derived from passphrase.
func EncryptStringE(data string, passphrase string) (string, error) {
	b, err := EncryptE([]byte(data), passphrase)
	return string(b), err
}

// DecryptStringE decrypts data produced by [EncryptStringE].
func DecryptStringE(data string, passphrase string) (string, error) {
	b, err := DecryptE([]byte(data), passphrase)
	return string(b), err
}

// EncryptE encrypts data with AES-GCM using a key derived from passphrase.
func EncryptE(data []byte, passphrase string) ([]byte, error) {
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

// DecryptE decrypts data produced by [EncryptE]. It returns an error (never
// panics) when the passphrase is wrong or the data is truncated or corrupt.
func DecryptE(data []byte, passphrase string) ([]byte, error) {
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

// EncryptString encrypts data. On failure it logs the error and returns "".
//
// Deprecated: use [EncryptStringE], which returns the error.
func EncryptString(data string, passphrase string) string {
	out, err := EncryptStringE(data, passphrase)
	if err != nil {
		slog.Error("encrypt failed", "err", err)
		return ""
	}
	return out
}

// DecryptString decrypts data. On failure it logs the error and returns "".
//
// Deprecated: use [DecryptStringE], which returns the error.
func DecryptString(data string, passphrase string) string {
	out, err := DecryptStringE(data, passphrase)
	if err != nil {
		slog.Error("decrypt failed", "err", err)
		return ""
	}
	return out
}

// Encrypt encrypts data. On failure it logs the error and returns nil.
//
// Deprecated: use [EncryptE], which returns the error.
func Encrypt(data []byte, passphrase string) []byte {
	out, err := EncryptE(data, passphrase)
	if err != nil {
		slog.Error("encrypt failed", "err", err)
		return nil
	}
	return out
}

// Decrypt decrypts data. On failure it logs the error and returns nil.
//
// Deprecated: use [DecryptE], which returns the error.
func Decrypt(data []byte, passphrase string) []byte {
	out, err := DecryptE(data, passphrase)
	if err != nil {
		slog.Error("decrypt failed", "err", err)
		return nil
	}
	return out
}

func EncryptFile(filename string, data []byte, passphrase string) {
	f, _ := os.Create(filename)
	defer f.Close()
	f.Write(Encrypt(data, passphrase))
}

func DecryptFile(filename string, passphrase string) []byte {
	data, _ := os.ReadFile(filename)
	return Decrypt(data, passphrase)
}
