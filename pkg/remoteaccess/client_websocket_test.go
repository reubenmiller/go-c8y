package remoteaccess

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// raServer accepts the remote access websocket and records its upgrade requests.
type raServer struct {
	*httptest.Server
	mu      sync.Mutex
	headers []http.Header
	paths   []string
	hits    atomic.Int64
}

func newRAServer(t *testing.T, useTLS bool) *raServer {
	t.Helper()
	s := &raServer{}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		if !websocket.IsWebSocketUpgrade(r) {
			http.NotFound(w, r)
			return
		}
		s.mu.Lock()
		s.headers = append(s.headers, r.Header.Clone())
		s.paths = append(s.paths, r.URL.Path)
		s.mu.Unlock()
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	})
	if useTLS {
		s.Server = httptest.NewTLSServer(handler)
	} else {
		s.Server = httptest.NewServer(handler)
	}
	t.Cleanup(s.Close)
	return s
}

func (s *raServer) Headers() []http.Header {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]http.Header(nil), s.headers...)
}

func raBasicAuth() authentication.AuthOptions {
	return authentication.AuthOptions{
		Tenant: "t123", Username: "user", Password: "secret",
		AuthType: []authentication.AuthType{authentication.AuthTypeBasic},
	}
}

func raClient(url string, auth authentication.AuthOptions, insecure bool, mw ...api.Middleware) *RemoteAccessClient {
	client := api.NewClient(api.ClientOptions{BaseURL: url, Auth: auth, InsecureSkipVerify: insecure, SkipTenantLookup: true, Middleware: mw})
	return NewRemoteAccessClient(client, RemoteAccessOptions{ManagedObjectID: "12345", RemoteAccessID: "1"})
}

func TestRemoteAccessDialTLS(t *testing.T) {
	srv := newRAServer(t, true)

	strict := raClient(srv.URL, raBasicAuth(), false)
	_, err := strict.DialContext(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "certificate")

	insecure := raClient(srv.URL, raBasicAuth(), true)
	conn, err := insecure.DialContext(context.Background())
	require.NoError(t, err)
	conn.Close()
	assert.Equal(t, []string{"/service/remoteaccess/client/12345/configurations/1"}, srv.paths)
}

func TestRemoteAccessDialAuthorizationAndMiddleware(t *testing.T) {
	srv := newRAServer(t, false)

	var current atomic.Value
	current.Store("token-1")
	source := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		return &authentication.Token{AccessToken: current.Load().(string)}, nil
	})
	trace := func(next http.RoundTripper) http.RoundTripper {
		return api.RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.Header.Set("X-Trace", "ra")
			return next.RoundTrip(r)
		})
	}
	ra := raClient(srv.URL, authentication.AuthOptions{TokenSource: source}, false, trace)

	for _, token := range []string{"token-1", "token-2"} {
		current.Store(token)
		conn, err := ra.DialContext(context.Background())
		require.NoError(t, err)
		conn.Close()
	}
	hs := srv.Headers()
	require.Len(t, hs, 2)
	assert.Equal(t, "Bearer token-1", hs[0].Get("Authorization"))
	assert.Equal(t, "Bearer token-2", hs[1].Get("Authorization"), "a refreshed token is used for the next connection")
	assert.Equal(t, "ra", hs[1].Get("X-Trace"), "middleware headers reach the upgrade request")
	assert.Equal(t, "application/json", hs[1].Get("Content-Type"))

	// the tenant override and extra headers of the context
	ctx := api.WithExtraHeaders(api.WithTenant(context.Background(), "t999"), map[string]string{"X-Extra": "1"})
	basic := raClient(srv.URL, raBasicAuth(), false)
	conn, err := basic.DialContext(ctx)
	require.NoError(t, err)
	conn.Close()
	last := srv.Headers()[2]
	assert.Equal(t, api.NewBasicAuthString("t999", "user", "secret"), last.Get("Authorization"))
	assert.Equal(t, "1", last.Get("X-Extra"))
}

func TestRemoteAccessContextCancel(t *testing.T) {
	release := make(chan struct{})
	hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer hang.Close()
	defer close(release)
	ra := raClient(hang.URL, raBasicAuth(), false)

	t.Run("DialContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()
		started := time.Now()
		_, err := ra.DialContext(ctx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled), err)
		assert.Less(t, time.Since(started), 5*time.Second)
	})

	t.Run("ListenServeContext", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		r, _ := io.Pipe()
		err := ra.ListenServeContext(ctx, r, io.Discard)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), err)
	})

	t.Run("ServeContext stops when ctx is cancelled", func(t *testing.T) {
		ra := raClient(hang.URL, raBasicAuth(), false)
		require.NoError(t, ra.Listen("127.0.0.1:0"))
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- ra.ServeContext(ctx) }()
		time.Sleep(100 * time.Millisecond)
		cancel()
		select {
		case err := <-done:
			assert.True(t, errors.Is(err, context.Canceled), err)
		case <-time.After(5 * time.Second):
			t.Fatal("ServeContext did not stop")
		}
	})
}

func TestRemoteAccessDryRun(t *testing.T) {
	srv := newRAServer(t, false)
	ra := raClient(srv.URL, raBasicAuth(), false)

	var handled []*http.Request
	ctx := api.WithDryRunHandler(api.WithDryRun(context.Background(), true), func(r *http.Request) { handled = append(handled, r) })

	check := func(t *testing.T, req *http.Request) {
		t.Helper()
		require.NotNil(t, req)
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/service/remoteaccess/client/12345/configurations/1", req.URL.Path)
		assert.Equal(t, "websocket", req.Header.Get("Upgrade"))
		assert.Equal(t, "Upgrade", req.Header.Get("Connection"))
		assert.Equal(t, "13", req.Header.Get("Sec-WebSocket-Version"))
		assert.Equal(t, api.NewBasicAuthString("t123", "user", "secret"), req.Header.Get("Authorization"))
	}

	t.Run("DialContext", func(t *testing.T) {
		conn, err := ra.DialContext(ctx)
		assert.Nil(t, conn)
		require.True(t, errors.Is(err, wsdial.ErrDryRun))
		check(t, wsdial.DryRunRequest(err))
		check(t, ra.DryRunRequest())
	})

	t.Run("ListenServeContext", func(t *testing.T) {
		r, _ := io.Pipe()
		assert.NoError(t, ra.ListenServeContext(ctx, r, io.Discard))
	})

	t.Run("ServeContext does not accept connections", func(t *testing.T) {
		require.NoError(t, ra.Listen("127.0.0.1:0"))
		assert.NoError(t, ra.ServeContext(ctx))
	})

	assert.Len(t, handled, 3)
	assert.Zero(t, srv.hits.Load(), "nothing is connected")

	t.Run("without a handler the request is logged with redacted headers", func(t *testing.T) {
		var logs bytes.Buffer
		old := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
		defer slog.SetDefault(old)

		ra := raClient(srv.URL, raBasicAuth(), false)
		_, err := ra.DialContext(api.WithDryRun(context.Background(), true))
		require.True(t, errors.Is(err, wsdial.ErrDryRun))
		assert.Contains(t, logs.String(), "DRY RUN")
		assert.Contains(t, logs.String(), "REDACTED")
		assert.NotContains(t, logs.String(), "c2VjcmV0")
		assert.Zero(t, srv.hits.Load())
	})
}
