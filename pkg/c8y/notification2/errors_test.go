package notification2

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const badHost = "http://exa mple.com"

func Test_InvalidHost_ReturnsError(t *testing.T) {
	_, err := NewClient(badHost, nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.Error(t, err)

	// no client (and no goroutine) is created for an invalid host
	c, err := NewClient(badHost, nil, Subscription{Token: "t"}, ConnectionOptions{})
	assert.Nil(t, c)
	assert.Error(t, err)
}

func Test_MissingDialer_ReturnsError(t *testing.T) {
	c, err := NewClient("http://localhost:1", nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.NoError(t, err)
	defer c.Close()
	c.dialer = nil
	assert.NotPanics(t, func() {
		err = c.Connect()
	})
	assert.True(t, errors.Is(err, ErrMissingDialer))
}

func Test_DialFailure_ReturnsError(t *testing.T) {
	c, err := NewClient("http://127.0.0.1:1", nil, Subscription{Token: "t"}, ConnectionOptions{})
	require.NoError(t, err)
	defer c.Close()
	assert.Error(t, c.Connect())
}
