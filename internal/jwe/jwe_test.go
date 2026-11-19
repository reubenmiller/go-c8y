package jwe

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func newKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	k, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestRoundTrip(t *testing.T) {
	key := newKey(t)
	token, err := Encrypt([]byte("header.payload.signature"), &key.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(token, key)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "header.payload.signature" {
		t.Fatalf("got %q", got)
	}
}

func TestWrongKey(t *testing.T) {
	token, _ := Encrypt([]byte("x"), &newKey(t).PublicKey)
	if _, err := Decrypt(token, newKey(t)); err == nil {
		t.Fatal("expected an error with the wrong key")
	}
}

func TestTampered(t *testing.T) {
	key := newKey(t)
	token, _ := Encrypt([]byte("secret"), &key.PublicKey)
	parts := strings.Split(token, ".")
	ct, _ := base64.RawURLEncoding.DecodeString(parts[3])
	ct[0] ^= 0xff
	parts[3] = base64.RawURLEncoding.EncodeToString(ct)
	if _, err := Decrypt(strings.Join(parts, "."), key); err == nil {
		t.Fatal("expected a decryption error for a tampered token")
	}
}

func TestUnsupportedAlgorithm(t *testing.T) {
	key := newKey(t)
	token, _ := Encrypt([]byte("x"), &key.PublicKey)
	parts := strings.Split(token, ".")
	parts[0] = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"RSA1_5","enc":"A256GCM"}`))
	_, err := Decrypt(strings.Join(parts, "."), key)
	if !errors.Is(err, ErrUnsupportedAlgorithm) || !strings.Contains(err.Error(), "RSA1_5") {
		t.Fatalf("err = %v, want ErrUnsupportedAlgorithm naming RSA1_5", err)
	}
}

func TestMalformed(t *testing.T) {
	if _, err := Decrypt("a.b.c", newKey(t)); err == nil {
		t.Fatal("expected an error for a malformed token")
	}
}
