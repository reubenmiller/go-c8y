package encrypt

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RoundTrip(t *testing.T) {
	enc, err := EncryptE([]byte("secret"), "pass")
	require.NoError(t, err)
	dec, err := DecryptE(enc, "pass")
	require.NoError(t, err)
	assert.Equal(t, "secret", string(dec))

	s, err := EncryptStringE("secret", "pass")
	require.NoError(t, err)
	d, err := DecryptStringE(s, "pass")
	require.NoError(t, err)
	assert.Equal(t, "secret", d)
}

func Test_DecryptFailuresReturnErrors(t *testing.T) {
	enc, err := EncryptE([]byte("secret"), "pass")
	require.NoError(t, err)

	_, err = DecryptE(enc, "wrong")
	assert.Error(t, err)

	_, err = DecryptE([]byte("short"), "pass")
	assert.True(t, errors.Is(err, ErrCiphertextTooShort))

	corrupt := append([]byte{}, enc...)
	corrupt[len(corrupt)-1] ^= 0xff
	_, err = DecryptE(corrupt, "pass")
	assert.Error(t, err)

	// the deprecated functions no longer panic
	assert.NotPanics(t, func() {
		assert.Nil(t, Decrypt([]byte("short"), "pass"))
		assert.Equal(t, "", DecryptString("short", "pass"))
		assert.Nil(t, Decrypt(enc, "wrong"))
	})
}
