package encrypt

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RoundTrip(t *testing.T) {
	enc, err := Encrypt([]byte("secret"), "pass")
	require.NoError(t, err)
	dec, err := Decrypt(enc, "pass")
	require.NoError(t, err)
	assert.Equal(t, "secret", string(dec))

	s, err := EncryptString("secret", "pass")
	require.NoError(t, err)
	d, err := DecryptString(s, "pass")
	require.NoError(t, err)
	assert.Equal(t, "secret", d)
}

func Test_DecryptFailuresReturnErrors(t *testing.T) {
	enc, err := Encrypt([]byte("secret"), "pass")
	require.NoError(t, err)

	_, err = Decrypt(enc, "wrong")
	assert.Error(t, err)

	_, err = Decrypt([]byte("short"), "pass")
	assert.True(t, errors.Is(err, ErrCiphertextTooShort))

	corrupt := append([]byte{}, enc...)
	corrupt[len(corrupt)-1] ^= 0xff
	_, err = Decrypt(corrupt, "pass")
	assert.Error(t, err)
}

func Test_FileRoundTrip(t *testing.T) {
	name := filepath.Join(t.TempDir(), "secret.bin")
	require.NoError(t, EncryptFile(name, []byte("secret"), "pass"))
	dec, err := DecryptFile(name, "pass")
	require.NoError(t, err)
	assert.Equal(t, "secret", string(dec))

	_, err = DecryptFile(name, "wrong")
	assert.Error(t, err)
	_, err = DecryptFile(filepath.Join(t.TempDir(), "missing"), "pass")
	assert.Error(t, err)
}
