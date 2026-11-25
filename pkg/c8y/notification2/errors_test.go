package notification2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const badHost = "http://exa mple.com"

func Test_InvalidHost_ReturnsError(t *testing.T) {
	_, err := NewNotification2ClientE(badHost, nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.Error(t, err)

	// the deprecated constructor must not exit the process
	c := NewNotification2Client(badHost, nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.NotNil(t, c)
	assert.Error(t, c.Connect())
	assert.NotPanics(t, func() { _ = c.URL(true); _ = c.Endpoint(); _ = c.Close() })
}

func Test_MissingDialer_ReturnsError(t *testing.T) {
	c, err := NewNotification2ClientE("http://localhost:1", nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.NoError(t, err)
	defer c.Close()
	c.dialer = nil
	assert.NotPanics(t, func() {
		err = c.Connect()
	})
	assert.True(t, errors.Is(err, ErrMissingDialer))
}

func Test_DialFailure_ReturnsError(t *testing.T) {
	c, err := NewNotification2ClientE("http://127.0.0.1:1", nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.NoError(t, err)
	defer c.Close()
	assert.Error(t, c.Connect())
}
