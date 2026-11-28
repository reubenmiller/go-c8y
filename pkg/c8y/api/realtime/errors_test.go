package realtime

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_InvalidHost_ReturnsError(t *testing.T) {
	_, err := NewClient(nil, ClientOptions{Host: "http://exa mple.com"})
	require.Error(t, err)

	// no client (and no goroutine) is created for an invalid host
	c, err := NewClient(nil, ClientOptions{Host: "http://exa mple.com"})
	assert.Nil(t, c)
	assert.Error(t, err)
}

func Test_MissingDialer_ReturnsError(t *testing.T) {
	c, err := NewClient(nil, ClientOptions{Host: "http://localhost:1"})
	require.NoError(t, err)
	c.dialer = nil
	assert.NotPanics(t, func() { err = c.Connect() })
	assert.True(t, errors.Is(err, ErrMissingDialer))
}

func Test_DialFailure_ReturnsError(t *testing.T) {
	c, err := NewClient(nil, ClientOptions{Host: "http://127.0.0.1:1"})
	require.NoError(t, err)
	assert.NotPanics(t, func() { err = c.Connect() })
	assert.Error(t, err)
}

// newBayeuxServer starts a websocket server which answers every handshake with reply.
func newBayeuxServer(t *testing.T, handshakeReply string) string {
	t.Helper()
	up := websocket.Upgrader{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer ws.Close()
		for {
			_, msg, err := ws.ReadMessage()
			if err != nil {
				return
			}
			if strings.Contains(string(msg), "/meta/handshake") {
				_ = ws.WriteMessage(websocket.TextMessage, []byte(handshakeReply))
			}
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func Test_RejectedHandshake_ReturnsErrorAndReleasesSubscribers(t *testing.T) {
	host := newBayeuxServer(t, `[{"channel":"/meta/handshake","id":"1","successful":false}]`)
	c, err := NewClient(nil, ClientOptions{Host: host, Tenant: "t", Username: "u", Password: "bad"})
	require.NoError(t, err)

	// a subscriber registered before the failure is released when it happens
	out := make(chan *Message, 1)
	c.hub.register <- &subscription{glob: mustGlob(t, "/alarms/*"), out: out, isWildcard: true}

	done := make(chan error, 1)
	go func() { done <- c.Connect() }()

	select {
	case err := <-done:
		assert.True(t, errors.Is(err, ErrHandshakeFailed), "got %v", err)
	case <-time.After(15 * time.Second):
		t.Fatal("Connect did not return")
	}
	assert.True(t, errors.Is(c.Err(), ErrHandshakeFailed))

	select {
	case _, ok := <-out:
		assert.False(t, ok, "subscriber channel should be closed")
	case <-time.After(10 * time.Second):
		t.Fatal("subscriber was not released")
	}
	assert.NoError(t, c.Close())
}

func Test_ServerAdviceNone_StopsClientWithoutPanic(t *testing.T) {
	host := newBayeuxServer(t, `[{"channel":"/meta/handshake","id":"1","successful":true,"clientId":"abc"},{"channel":"/meta/connect","successful":true,"advice":{"reconnect":"none"}}]`)
	c, err := NewClient(nil, ClientOptions{Host: host, Tenant: "t", Username: "u", Password: "p"})
	require.NoError(t, err)
	_ = c.Connect()
	assert.Eventually(t, func() bool { return errors.Is(c.Err(), ErrServerNoReconnect) }, 10*time.Second, 50*time.Millisecond)
	_ = c.Close()
}

func mustGlob(t *testing.T, pattern string) *matcher.Glob {
	t.Helper()
	g, err := matcher.Compile(pattern)
	require.NoError(t, err)
	return g
}

func Test_DialFailure_ReturnsConnectError(t *testing.T) {
	c, err := NewClient(nil, ClientOptions{Host: "http://127.0.0.1:1"})
	require.NoError(t, err)
	err = c.Connect()
	require.Error(t, err)

	var connErr *ConnectError
	require.True(t, errors.As(err, &connErr), "want a *ConnectError, got %T", err)
	assert.Equal(t, "ws://127.0.0.1:1/cep/realtime", connErr.URL)
	require.Error(t, connErr.Err)
	assert.Equal(t, errors.Unwrap(err), connErr.Err)
	assert.Equal(t, "realtime: failed to connect to ws://127.0.0.1:1/cep/realtime: "+connErr.Err.Error(), err.Error())
}

func Test_ConnectError_NotUsedForOtherErrors(t *testing.T) {
	c, err := NewClient(nil, ClientOptions{Host: "http://localhost:1"})
	require.NoError(t, err)
	c.dialer = nil
	err = c.Connect()
	var connErr *ConnectError
	assert.False(t, errors.As(err, &connErr))
	assert.True(t, errors.Is(err, ErrMissingDialer))
}

func Test_ConnectError_UnwrapsCause(t *testing.T) {
	cause := errors.New("boom")
	err := error(&ConnectError{URL: "ws://h/cep/realtime", Err: cause})
	assert.True(t, errors.Is(err, cause))
	assert.Equal(t, "realtime: failed to connect to ws://h/cep/realtime: boom", err.Error())
}
