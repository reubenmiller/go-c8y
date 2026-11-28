package api

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/notification2"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// wsRecord is what a test websocket server saw of one upgrade request.
type wsRecord struct {
	Path   string
	Query  string
	Header http.Header
}

// wsTestServer accepts websocket upgrades on every path under /cep/realtime
// and /notification2/consumer and /service/remoteaccess; other requests are
// served by other. It records the upgrade requests, answers Bayeux messages on
// the realtime path and otherwise keeps the connection open.
type wsTestServer struct {
	*httptest.Server

	mu         sync.Mutex
	records    []wsRecord
	handshakes []json.RawMessage // ext of every /meta/handshake
	other      http.HandlerFunc
	hits       atomic.Int64
	last       *websocket.Conn // the last realtime connection
	writeMu    sync.Mutex
}

// Push sends a Bayeux message array on the last realtime connection.
func (s *wsTestServer) Push(t *testing.T, messages ...map[string]any) {
	t.Helper()
	s.mu.Lock()
	conn := s.last
	s.mu.Unlock()
	require.NotNil(t, conn)
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	require.NoError(t, conn.WriteJSON(messages))
}

func newWSTestServer(t *testing.T, useTLS bool, other http.HandlerFunc, configure ...func(*httptest.Server)) *wsTestServer {
	t.Helper()
	s := &wsTestServer{other: other}
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.hits.Add(1)
		if !websocket.IsWebSocketUpgrade(r) {
			if s.other != nil {
				s.other(w, r)
				return
			}
			http.NotFound(w, r)
			return
		}
		s.mu.Lock()
		s.records = append(s.records, wsRecord{Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone()})
		s.mu.Unlock()

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if strings.HasPrefix(r.URL.Path, "/cep/realtime") {
			s.mu.Lock()
			s.last = conn
			s.mu.Unlock()
		}
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if !strings.HasPrefix(r.URL.Path, "/cep/realtime") {
				continue
			}
			var msgs []map[string]json.RawMessage
			if json.Unmarshal(data, &msgs) != nil {
				continue
			}
			var replies []map[string]any
			for _, m := range msgs {
				var channel, id string
				_ = json.Unmarshal(m["channel"], &channel)
				_ = json.Unmarshal(m["id"], &id)
				if channel == "/meta/connect" && m["advice"] == nil {
					continue // long poll: a real server holds this reply
				}
				reply := map[string]any{"channel": channel, "id": id, "successful": true, "clientId": "client-1"}
				switch channel {
				case "/meta/handshake":
					s.mu.Lock()
					s.handshakes = append(s.handshakes, m["ext"])
					s.mu.Unlock()
				case "/meta/subscribe":
					var sub string
					_ = json.Unmarshal(m["subscription"], &sub)
					reply["subscription"] = sub
				}
				replies = append(replies, reply)
			}
			if len(replies) > 0 {
				s.writeMu.Lock()
				_ = conn.WriteJSON(replies)
				s.writeMu.Unlock()
			}
		}
	})
	s.Server = httptest.NewUnstartedServer(handler)
	for _, fn := range configure {
		fn(s.Server)
	}
	if useTLS {
		s.StartTLS()
	} else {
		s.Start()
	}
	t.Cleanup(s.Close)
	return s
}

func (s *wsTestServer) Records() []wsRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]wsRecord(nil), s.records...)
}

func (s *wsTestServer) Handshakes() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]string, len(s.handshakes))
	for i, h := range s.handshakes {
		out[i] = string(h)
	}
	return out
}

// wsHangingServer accepts TCP connections and never answers the handshake.
func wsHangingServer(t *testing.T) *httptest.Server {
	t.Helper()
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() {
		close(release)
		srv.Close()
	})
	return srv
}

func wsTestClient(url string, auth authentication.AuthOptions, insecure bool, mw ...Middleware) *Client {
	return mustNewClient(ClientOptions{
		BaseURL:            url,
		Auth:               auth,
		InsecureSkipVerify: insecure,
		SkipTenantLookup:   true,
		Middleware:         mw,
	})
}

func wsBasicAuth() authentication.AuthOptions {
	return authentication.AuthOptions{Tenant: "t123", Username: "user", Password: "secret", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}}
}

func TestWebsocketTLSInsecureSkipVerify(t *testing.T) {
	srv := newWSTestServer(t, true, nil)

	t.Run("verification on rejects the test certificate", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), false)
		_, _, err := client.DialWebsocket(context.Background(), srv.URL+"/service/remoteaccess/x", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})

	t.Run("InsecureSkipVerify is respected by the dialer", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), true)
		assert.True(t, client.WebsocketDialer().TLSClientConfig.InsecureSkipVerify)
		conn, resp, err := client.DialWebsocket(context.Background(), srv.URL+"/service/remoteaccess/x", nil)
		require.NoError(t, err)
		defer conn.Close()
		assert.Equal(t, "/service/remoteaccess/x", resp.Request.URL.Path)
	})

	t.Run("after an HTTP request on a server which offers HTTP/2", func(t *testing.T) {
		// The HTTP transport adds h2 to its TLS config after its first request;
		// the websocket handshake must still use HTTP/1.1.
		h2 := newWSTestServer(t, true, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("{}"))
		}, func(s *httptest.Server) { s.EnableHTTP2 = true })
		client := wsTestClient(h2.URL, wsBasicAuth(), true)
		resp, err := client.Resty().R().Get("/inventory/managedObjects")
		require.NoError(t, err)
		assert.Equal(t, "HTTP/2.0", resp.Proto())

		conn, _, err := client.DialWebsocket(context.Background(), h2.URL+"/service/remoteaccess/x", nil)
		require.NoError(t, err)
		conn.Close()
	})

	t.Run("realtime connects with InsecureSkipVerify", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), true)
		rt := client.Realtime()
		require.NoError(t, rt.ConnectContext(context.Background()))
		defer rt.Close()
		assert.True(t, rt.IsConnected())
	})

	t.Run("realtime fails without InsecureSkipVerify", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), false)
		err := client.Realtime().ConnectContext(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})
}

func TestWebsocketDialerUsesTransportSettings(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "device"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	keyPEM := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))

	proxyURL, _ := url.Parse("http://proxy.example:3128")
	var proxied atomic.Int64
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = func(r *http.Request) (*url.URL, error) {
		proxied.Add(1)
		return proxyURL, nil
	}
	client := mustNewClient(ClientOptions{
		BaseURL:            "https://example.invalid",
		Transport:          transport,
		InsecureSkipVerify: true,
		SkipTenantLookup:   true,
		Auth:               authentication.AuthOptions{Tenant: "t1", Username: "u", Password: "p", AuthType: []authentication.AuthType{authentication.AuthTypeBasic}, Certificate: certPEM, CertificateKey: keyPEM},
	})
	dialer := client.WebsocketDialer()
	require.NotNil(t, dialer.TLSClientConfig)
	assert.True(t, dialer.TLSClientConfig.InsecureSkipVerify)
	// the proxy of the HTTP transport is used
	require.NotNil(t, dialer.Proxy)
	got, err := dialer.Proxy(&http.Request{URL: &url.URL{Scheme: "https", Host: "example.invalid"}})
	require.NoError(t, err)
	assert.Equal(t, "proxy.example:3128", got.Host)
	assert.Positive(t, proxied.Load())
	require.Len(t, dialer.TLSClientConfig.Certificates, 1, "the client certificate is used for mTLS")
}

func TestWebsocketHeaderAuthorization(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	ctx := context.Background()

	t.Run("basic auth with the tenant", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), false)
		h, err := client.WebsocketHeader(ctx)
		require.NoError(t, err)
		assert.Equal(t, NewBasicAuthString("t123", "user", "secret"), h.Get("Authorization"))
		assert.NotEmpty(t, h.Get("User-Agent"))
		for _, k := range []string{"Upgrade", "Connection", "Sec-Websocket-Version"} {
			assert.Empty(t, h.Get(k), k)
		}
	})

	t.Run("tenant override of the context", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), false)
		h, err := client.WebsocketHeader(WithTenant(ctx, "t999"))
		require.NoError(t, err)
		assert.Equal(t, NewBasicAuthString("t999", "user", "secret"), h.Get("Authorization"))
	})

	t.Run("credentials and headers of the context", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), false)
		c := WithAuth(ctx, authentication.AuthOptions{Token: "ctx-token"})
		c = WithExtraHeaders(c, map[string]string{"X-Extra": "1"})
		h, err := client.WebsocketHeader(c)
		require.NoError(t, err)
		assert.Equal(t, "Bearer ctx-token", h.Get("Authorization"))
		assert.Equal(t, "1", h.Get("X-Extra"))
	})

	t.Run("no auth", func(t *testing.T) {
		client := wsTestClient(srv.URL, wsBasicAuth(), false)
		h, err := client.WebsocketHeader(WithNoAuth(ctx))
		require.NoError(t, err)
		assert.Empty(t, h.Get("Authorization"))
	})

	t.Run("a refreshed token is used for the next connection", func(t *testing.T) {
		var current atomic.Value
		current.Store("token-1")
		source := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
			return &authentication.Token{AccessToken: current.Load().(string)}, nil
		})
		client := wsTestClient(srv.URL, authentication.AuthOptions{TokenSource: source}, false)

		for _, want := range []string{"token-1", "token-2"} {
			current.Store(want)
			conn, resp, err := client.DialWebsocket(ctx, srv.URL+"/service/remoteaccess/x", nil)
			require.NoError(t, err)
			conn.Close()
			assert.Equal(t, "Bearer "+want, resp.Request.Header.Get("Authorization"))
		}
		recs := srv.Records()
		require.GreaterOrEqual(t, len(recs), 2)
		assert.Equal(t, "Bearer token-2", recs[len(recs)-1].Header.Get("Authorization"))
	})

	t.Run("a token source error fails the connection", func(t *testing.T) {
		source := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
			return nil, errors.New("no token")
		})
		client := wsTestClient(srv.URL, authentication.AuthOptions{TokenSource: source}, false)
		_, _, err := client.DialWebsocket(ctx, srv.URL+"/service/remoteaccess/x", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no token")
	})
}

func TestWebsocketMiddlewareHeaders(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	var seenURL, seenMethod string
	var responseSide atomic.Int64
	addHeader := func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			seenURL, seenMethod = r.URL.String(), r.Method
			r.Header.Set("X-Trace", "abc")
			resp, err := next.RoundTrip(r)
			responseSide.Add(1)
			return resp, err
		})
	}
	client := wsTestClient(srv.URL, wsBasicAuth(), false, addHeader)
	client.Use(func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			r.Header.Set("X-Second", "2")
			return next.RoundTrip(r)
		})
	})

	conn, _, err := client.DialWebsocket(context.Background(), srv.URL+"/service/remoteaccess/x", nil)
	require.NoError(t, err)
	conn.Close()

	rec := srv.Records()[0]
	assert.Equal(t, "abc", rec.Header.Get("X-Trace"))
	assert.Equal(t, "2", rec.Header.Get("X-Second"))
	assert.Equal(t, NewBasicAuthString("t123", "user", "secret"), rec.Header.Get("Authorization"))
	assert.Equal(t, http.MethodGet, seenMethod)
	assert.True(t, strings.HasPrefix(seenURL, "http://"), seenURL)

	// realtime uses it too
	rt := client.Realtime()
	require.NoError(t, rt.ConnectContext(context.Background()))
	defer rt.Close()
	recs := srv.Records()
	last := recs[len(recs)-1]
	assert.Equal(t, "/cep/realtime", last.Path)
	assert.Equal(t, "abc", last.Header.Get("X-Trace"))

	// a middleware error fails the connection without connecting
	failing := wsTestClient(srv.URL, wsBasicAuth(), false, func(http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("blocked") })
	})
	before := srv.hits.Load()
	_, _, err = failing.DialWebsocket(context.Background(), srv.URL+"/service/remoteaccess/x", nil)
	require.ErrorContains(t, err, "blocked")
	assert.Equal(t, before, srv.hits.Load())
}

func TestWebsocketContextCancelAbortsConnecting(t *testing.T) {
	hang := wsHangingServer(t)
	client := wsTestClient(hang.URL, wsBasicAuth(), false)

	t.Run("DialWebsocket", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		defer cancel()
		started := time.Now()
		_, _, err := client.DialWebsocket(ctx, hang.URL+"/service/remoteaccess/x", nil)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.DeadlineExceeded), err)
		assert.Less(t, time.Since(started), 5*time.Second)
	})

	t.Run("realtime ConnectContext", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(200 * time.Millisecond)
			cancel()
		}()
		started := time.Now()
		err := client.Realtime().ConnectContext(ctx)
		require.Error(t, err)
		assert.True(t, errors.Is(err, context.Canceled), err)
		assert.Less(t, time.Since(started), 5*time.Second)
	})
}

func TestWebsocketDryRun(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	client := wsTestClient(srv.URL, wsBasicAuth(), false)

	t.Run("DialWebsocket reports the upgrade request", func(t *testing.T) {
		var handled []*http.Request
		ctx := WithDryRunHandler(WithDryRun(context.Background(), true), func(r *http.Request) { handled = append(handled, r) })
		conn, resp, err := client.DialWebsocket(ctx, srv.URL+"/notification2/consumer/?token=secret-token&consumer=c1", nil)
		require.Error(t, err)
		assert.Nil(t, conn)
		assert.Nil(t, resp)
		assert.True(t, errors.Is(err, wsdial.ErrDryRun))

		req := wsdial.DryRunRequest(err)
		require.NotNil(t, req)
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "websocket", req.Header.Get("Upgrade"))
		assert.Equal(t, "Upgrade", req.Header.Get("Connection"))
		assert.Equal(t, "13", req.Header.Get("Sec-WebSocket-Version"))
		assert.Equal(t, NewBasicAuthString("t123", "user", "secret"), req.Header.Get("Authorization"), "same as an HTTP dry run request")
		assert.NotContains(t, req.URL.String(), "secret-token")
		assert.Equal(t, "c1", req.URL.Query().Get("consumer"))
		require.Len(t, handled, 1)
		assert.Equal(t, req.URL.String(), handled[0].URL.String())
		assert.Zero(t, srv.hits.Load(), "nothing is connected")
	})

	t.Run("without a handler the request is logged with redacted headers", func(t *testing.T) {
		var buf bytes.Buffer
		old := slog.Default()
		slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
		defer slog.SetDefault(old)

		_, _, err := client.DialWebsocket(WithDryRun(context.Background(), true), srv.URL+"/service/remoteaccess/x", nil)
		require.True(t, errors.Is(err, wsdial.ErrDryRun))
		out := buf.String()
		assert.Contains(t, out, "DRY RUN")
		assert.Contains(t, out, "method=GET")
		assert.Contains(t, out, "REDACTED")
		assert.NotContains(t, out, "c2VjcmV0")
		assert.Zero(t, srv.hits.Load())
	})

	t.Run("a token source is not called", func(t *testing.T) {
		var calls atomic.Int64
		source := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
			calls.Add(1)
			return &authentication.Token{AccessToken: "t"}, nil
		})
		c := wsTestClient(srv.URL, authentication.AuthOptions{TokenSource: source}, false)
		before := calls.Load()
		_, _, err := c.DialWebsocket(WithDryRun(context.Background(), true), srv.URL+"/service/remoteaccess/x", nil)
		require.True(t, errors.Is(err, wsdial.ErrDryRun))
		assert.Equal(t, before, calls.Load())
	})
}

func TestRealtimeDryRunSubscribeStream(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	client := wsTestClient(srv.URL, wsBasicAuth(), false)

	var handled []*http.Request
	ctx := WithDryRunHandler(WithDryRun(context.Background(), true), func(r *http.Request) { handled = append(handled, r) })

	check := func(t *testing.T, status string, err error, req *http.Request, meta map[string]any, channels ...string) {
		t.Helper()
		require.NoError(t, err)
		assert.Equal(t, status, "Skipped")
		require.NotNil(t, req)
		assert.Equal(t, http.MethodGet, req.Method)
		assert.Equal(t, "/cep/realtime", req.URL.Path)
		assert.Equal(t, "websocket", req.Header.Get("Upgrade"))
		assert.Equal(t, channels, meta[realtime.DryRunMetaChannels])
	}

	t.Run("alarms", func(t *testing.T) {
		res := client.Alarms.SubscribeStream(ctx, "123")
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/alarms/123")
		assert.False(t, res.Data.Next())
		assert.NoError(t, res.Data.Err())
		assert.NoError(t, res.Data.Close())
	})
	t.Run("events", func(t *testing.T) {
		res := client.Events.SubscribeStream(ctx, "123")
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/events/123")
		for range res.Data.Seq() {
			t.Fatal("dry run streams are empty")
		}
	})
	t.Run("operations", func(t *testing.T) {
		res := client.Operations.SubscribeStream(ctx, "123")
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/operations/123")
	})
	t.Run("measurements", func(t *testing.T) {
		res := client.Measurements.SubscribeStream(ctx, "123")
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/measurements/123")
	})
	t.Run("managed objects", func(t *testing.T) {
		res := client.ManagedObjects.SubscribeStream(ctx, "123")
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/managedobjects/123")
	})
	t.Run("several channels", func(t *testing.T) {
		res := client.Realtime().SubscribeStream(ctx, realtime.Alarms("1"), realtime.Events("*"))
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/alarms/1", "/events/*")
		assert.False(t, res.Data.Next())
	})
	t.Run("Subscribe", func(t *testing.T) {
		out := make(chan *realtime.Message, 1)
		res := client.Alarms.Subscribe(ctx, "123", out)
		check(t, string(res.Status), res.Err, res.Request, res.Meta, "/alarms/123")
		_, open := <-out
		assert.False(t, open, "out is closed")
	})

	assert.Len(t, handled, 7, "one upgrade request per dry run")
	assert.Zero(t, srv.hits.Load(), "nothing is connected")
	assert.False(t, client.Realtime().IsConnected())
}

func TestRealtimeUsesCurrentAuthorization(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	var current atomic.Value
	current.Store("token-1")
	source := authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		return &authentication.Token{AccessToken: current.Load().(string)}, nil
	})
	client := wsTestClient(srv.URL, authentication.AuthOptions{TokenSource: source}, false)
	rt := client.Realtime()

	require.NoError(t, rt.ConnectContext(context.Background()))
	require.NoError(t, rt.Close())

	// The token is refreshed after the realtime client was created and used
	current.Store("token-2")
	require.NoError(t, rt.ConnectContext(context.Background()))
	defer rt.Close()

	recs := srv.Records()
	require.Len(t, recs, 2)
	assert.Equal(t, "Bearer token-1", recs[0].Header.Get("Authorization"))
	assert.Equal(t, "Bearer token-2", recs[1].Header.Get("Authorization"))

	hs := srv.Handshakes()
	require.Len(t, hs, 2)
	assert.Contains(t, hs[0], `"token-1"`)
	assert.Contains(t, hs[1], `"token-2"`)
}

// The server can ask for a new handshake on a connection which is kept. The
// token of the connection may have expired by then, so the handshake must use
// the current authorization of the client.
func TestRealtimeRehandshakeUsesCurrentAuthorization(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	var current atomic.Value
	current.Store("token-1")
	// a caching source: the cached token is dropped for the new handshake
	source := authentication.NewCachedTokenSource(authentication.TokenSourceFunc(func() (*authentication.Token, error) {
		return &authentication.Token{AccessToken: current.Load().(string)}, nil
	}))
	client := wsTestClient(srv.URL, authentication.AuthOptions{TokenSource: source}, false)
	rt := client.Realtime()
	require.NoError(t, rt.ConnectContext(context.Background()))
	defer rt.Close()
	require.Len(t, srv.Handshakes(), 1)

	current.Store("token-2")
	srv.Push(t, map[string]any{
		"channel": "/meta/connect", "successful": false,
		"advice": map[string]any{"reconnect": "handshake", "interval": 10},
	})
	require.Eventually(t, func() bool { return len(srv.Handshakes()) == 2 }, 5*time.Second, 20*time.Millisecond)
	assert.Contains(t, srv.Handshakes()[0], `"token-1"`)
	assert.Contains(t, srv.Handshakes()[1], `"token-2"`)
}

func TestRealtimeBasicAuthHandshake(t *testing.T) {
	srv := newWSTestServer(t, false, nil)
	client := wsTestClient(srv.URL, wsBasicAuth(), false)
	rt := client.Realtime()
	require.NoError(t, rt.ConnectContext(WithTenant(context.Background(), "t555")))
	defer rt.Close()

	hs := srv.Handshakes()
	require.Len(t, hs, 1)
	// the Bayeux token is the base64 of "tenant/user:password" of the connection
	assert.Contains(t, hs[0], "dDU1NS91c2VyOnNlY3JldA==") // t555/user:secret
}

func TestNotification2UsesClientSettings(t *testing.T) {
	srv := newWSTestServer(t, true, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/notification2/token" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"token":"n2-token"}`))
			return
		}
		http.NotFound(w, r)
	})
	var requests atomic.Int64
	client := wsTestClient(srv.URL, wsBasicAuth(), true, func(next http.RoundTripper) http.RoundTripper {
		return RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
			requests.Add(1)
			r.Header.Set("X-Trace", "n2")
			return next.RoundTrip(r)
		})
	})
	opt := notification2.ClientOptions{Consumer: "my-consumer", Options: notification2.TokenOptions{Subscription: "sub"}}

	t.Run("connects with the TLS settings and middleware headers", func(t *testing.T) {
		res := client.Notification2.SubscribeStream(context.Background(), opt)
		require.NoError(t, res.Err)
		defer res.Data.Close()

		recs := srv.Records()
		require.Len(t, recs, 1)
		assert.Equal(t, "/notification2/consumer/", recs[0].Path)
		assert.Contains(t, recs[0].Query, "token=n2-token")
		assert.Equal(t, "n2", recs[0].Header.Get("X-Trace"))
		assert.NotEmpty(t, recs[0].Header.Get("User-Agent"))
		assert.Empty(t, recs[0].Header.Get("Authorization"), "the token in the URL authenticates the connection")
	})

	t.Run("without InsecureSkipVerify the connection fails", func(t *testing.T) {
		strict := wsTestClient(srv.URL, wsBasicAuth(), false)
		cl, err := client.Notification2.CreateClient(context.Background(), opt)
		require.NoError(t, err)
		cl.SetConnector(strict)
		defer cl.Close()
		err = cl.ConnectContext(context.Background())
		require.Error(t, err)
		assert.Contains(t, err.Error(), "certificate")
	})

	t.Run("ctx cancel aborts connecting", func(t *testing.T) {
		release := make(chan struct{})
		hang := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/notification2/token" {
				_, _ = w.Write([]byte(`{"token":"n2-token"}`))
				return
			}
			<-release
		}))
		defer hang.Close()
		defer close(release)
		c := wsTestClient(hang.URL, wsBasicAuth(), false)
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()
		started := time.Now()
		res := c.Notification2.SubscribeStream(ctx, opt)
		require.Error(t, res.Err)
		assert.True(t, errors.Is(res.Err, context.DeadlineExceeded), res.Err)
		assert.Less(t, time.Since(started), 5*time.Second)
	})

	t.Run("dry run does not connect", func(t *testing.T) {
		before := len(srv.Records())
		var handled []*http.Request
		ctx := WithDryRunHandler(WithDryRun(context.Background(), true), func(r *http.Request) { handled = append(handled, r) })
		res := client.Notification2.SubscribeStream(ctx, opt)
		require.NoError(t, res.Err)
		assert.Equal(t, "Skipped", string(res.Status))
		require.NotNil(t, res.Request)
		assert.Equal(t, "/notification2/consumer/", res.Request.URL.Path)
		assert.Equal(t, "myconsumer", res.Meta["consumer"], "the cleaned consumer name")
		assert.NotContains(t, res.Request.URL.RawQuery, "n2-token")
		assert.NotContains(t, res.Meta["url"].(string), "n2-token")
		assert.Equal(t, "websocket", res.Request.Header.Get("Upgrade"))
		// the token request is a dry run HTTP request too, the upgrade request follows it
		require.GreaterOrEqual(t, len(handled), 2)
		assert.Equal(t, http.MethodPost, handled[0].Method)
		assert.Equal(t, res.Request.URL.String(), handled[len(handled)-1].URL.String())
		assert.False(t, res.Data.Next())
		assert.NoError(t, res.Data.Err())
		assert.Equal(t, before, len(srv.Records()), "no websocket connection")
	})
}
