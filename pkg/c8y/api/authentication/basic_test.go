package authentication

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_DecodeBasicAuth_InvalidInputReturnsError(t *testing.T) {
	// valid base64, but no "user:password" separator (used to panic)
	noColon := base64.StdEncoding.EncodeToString([]byte("justuser"))
	assert.NotPanics(t, func() {
		_, err := DecodeBasicAuth(noColon, "host")
		assert.Error(t, err)
	})
	// not base64 at all
	_, err := DecodeBasicAuth("%%%", "host")
	assert.Error(t, err)

	creds, err := DecodeBasicAuth(base64.StdEncoding.EncodeToString([]byte("t1/user:pass")), "host")
	require.NoError(t, err)
	assert.Equal(t, "t1", creds.Tenant)
	assert.Equal(t, "user", creds.Username)
	assert.Equal(t, "pass", creds.Password)
}
