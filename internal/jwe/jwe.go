// Package jwe decrypts and encrypts compact JWE tokens using the only
// algorithm pair Cumulocity uses for certificate access tokens:
// key management RSA-OAEP-256 and content encryption A256GCM.
package jwe

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	// AlgRSAOAEP256 is the supported key management algorithm.
	AlgRSAOAEP256 = "RSA-OAEP-256"
	// EncA256GCM is the supported content encryption algorithm.
	EncA256GCM = "A256GCM"
)

var b64 = base64.RawURLEncoding

type header struct {
	Alg string `json:"alg"`
	Enc string `json:"enc"`
}

// ErrUnsupportedAlgorithm is returned for tokens that do not use
// RSA-OAEP-256 with A256GCM.
var ErrUnsupportedAlgorithm = errors.New("unsupported JWE algorithm")

// Decrypt decrypts a compact JWE with the given RSA private key and returns
// the plaintext.
func Decrypt(compact string, key *rsa.PrivateKey) ([]byte, error) {
	if key == nil {
		return nil, errors.New("jwe: no private key")
	}
	parts := strings.Split(compact, ".")
	if len(parts) != 5 {
		return nil, fmt.Errorf("jwe: expected 5 parts, got %d", len(parts))
	}
	protected, encKey, iv, ciphertext, tag := parts[0], parts[1], parts[2], parts[3], parts[4]

	rawHeader, err := b64.DecodeString(protected)
	if err != nil {
		return nil, fmt.Errorf("jwe: header: %w", err)
	}
	var h header
	if err := json.Unmarshal(rawHeader, &h); err != nil {
		return nil, fmt.Errorf("jwe: header: %w", err)
	}
	if h.Alg != AlgRSAOAEP256 || h.Enc != EncA256GCM {
		return nil, fmt.Errorf("jwe: %w: alg=%q enc=%q (supported: alg=%q enc=%q)", ErrUnsupportedAlgorithm, h.Alg, h.Enc, AlgRSAOAEP256, EncA256GCM)
	}

	wrapped, err := b64.DecodeString(encKey)
	if err != nil {
		return nil, fmt.Errorf("jwe: encrypted key: %w", err)
	}
	cek, err := rsa.DecryptOAEP(sha256.New(), nil, key, wrapped, nil)
	if err != nil {
		return nil, fmt.Errorf("jwe: decrypt content key: %w", err)
	}
	if len(cek) != 32 {
		return nil, fmt.Errorf("jwe: content key is %d bytes, want 32", len(cek))
	}

	nonce, err := b64.DecodeString(iv)
	if err != nil {
		return nil, fmt.Errorf("jwe: iv: %w", err)
	}
	ct, err := b64.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("jwe: ciphertext: %w", err)
	}
	authTag, err := b64.DecodeString(tag)
	if err != nil {
		return nil, fmt.Errorf("jwe: tag: %w", err)
	}

	gcm, err := newGCM(cek, len(nonce))
	if err != nil {
		return nil, err
	}
	// The additional authenticated data is the ASCII protected header.
	plaintext, err := gcm.Open(nil, nonce, append(ct, authTag...), []byte(protected))
	if err != nil {
		return nil, fmt.Errorf("jwe: decrypt content: %w", err)
	}
	return plaintext, nil
}

// Encrypt encrypts plaintext to the RSA public key as a compact JWE. It is
// used by tests and the offline fake server to mimic the platform.
func Encrypt(plaintext []byte, pub *rsa.PublicKey) (string, error) {
	rawHeader, err := json.Marshal(header{Alg: AlgRSAOAEP256, Enc: EncA256GCM})
	if err != nil {
		return "", err
	}
	protected := b64.EncodeToString(rawHeader)

	cek := make([]byte, 32)
	if _, err := rand.Read(cek); err != nil {
		return "", err
	}
	wrapped, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, cek, nil)
	if err != nil {
		return "", fmt.Errorf("jwe: encrypt content key: %w", err)
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	gcm, err := newGCM(cek, len(nonce))
	if err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, plaintext, []byte(protected))
	ct, tag := sealed[:len(sealed)-gcm.Overhead()], sealed[len(sealed)-gcm.Overhead():]

	return strings.Join([]string{
		protected,
		b64.EncodeToString(wrapped),
		b64.EncodeToString(nonce),
		b64.EncodeToString(ct),
		b64.EncodeToString(tag),
	}, "."), nil
}

func newGCM(key []byte, nonceSize int) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("jwe: %w", err)
	}
	gcm, err := cipher.NewGCMWithNonceSize(block, nonceSize)
	if err != nil {
		return nil, fmt.Errorf("jwe: %w", err)
	}
	return gcm, nil
}
