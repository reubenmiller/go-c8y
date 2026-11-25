package api

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/wsdial"
)

// websocketHandshakeTimeout is the default time allowed to complete the
// websocket handshake (the same as gorilla's default dialer).
const websocketHandshakeTimeout = 45 * time.Second

// Client implements wsdial.Connector, so the realtime, notification2 and remote
// access clients can connect with the settings of this client.
var _ wsdial.Connector = (*Client)(nil)

// WebsocketDialer returns a websocket dialer which uses the connection settings
// of the HTTP client: the TLS configuration of its transport (including
// InsecureSkipVerify, extra root CAs and client certificates for mTLS) and its
// proxy and network dial function.
//
// A new dialer is returned for every call, built from the current transport
// settings, so changes made to the client after NewClient are picked up.
// Websocket clients that reconnect should call this (or [Client.DialWebsocket])
// for each connection attempt.
//
// Limits: only settings of the underlying *http.Transport are copied. A custom
// transport which is not an *http.Transport (and does not expose one with a
// BaseTransport method) falls back to the proxy from the environment, and its
// TLS settings are only used when it exposes TLSClientConfig().
func (c *Client) WebsocketDialer() *websocket.Dialer {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: websocketHandshakeTimeout,
	}

	var transports []http.RoundTripper
	if c.httpClient != nil {
		transports = append(transports, c.httpClient.Transport())
	}
	transports = append(transports, c.baseTransport)

	var tlsConfig *tls.Config
	foundTransport := false
	for _, rt := range transports {
		t, cfg := unwrapTransport(rt)
		if tlsConfig == nil && cfg != nil {
			tlsConfig = cfg
		}
		if t != nil && !foundTransport {
			foundTransport = true
			dialer.Proxy = t.Proxy
			dialer.NetDialContext = t.DialContext
		}
	}
	if tlsConfig != nil {
		dialer.TLSClientConfig = tlsConfig.Clone()
		// The HTTP transport adds "h2" to NextProtos once it has made a
		// request. A websocket needs HTTP/1.1: the server would otherwise
		// negotiate h2 and the handshake fails.
		dialer.TLSClientConfig.NextProtos = nil
	}
	return dialer
}

// unwrapTransport walks a transport chain (DryRunTransport, middleware, wrappers
// exposing BaseTransport) and returns the first *http.Transport and the TLS
// configuration found.
func unwrapTransport(rt http.RoundTripper) (*http.Transport, *tls.Config) {
	var cfg *tls.Config
	for i := 0; i < 16 && rt != nil; i++ {
		switch t := rt.(type) {
		case *http.Transport:
			if t.TLSClientConfig != nil {
				cfg = t.TLSClientConfig
			}
			return t, cfg
		case interface{ TLSClientConfig() *tls.Config }:
			if cfg == nil {
				cfg = t.TLSClientConfig()
			}
		}
		next, ok := rt.(interface{ BaseTransport() http.RoundTripper })
		if !ok {
			return nil, cfg
		}
		rt = next.BaseTransport()
	}
	return nil, cfg
}

// WebsocketHeader returns the request headers a websocket connection made with
// ctx should send, so it authenticates and identifies like an HTTP request of
// this client:
//
//   - Authorization from [Client.AuthorizationHeader]: per-request credentials
//     in ctx, the token source (refreshing an expired token), certificate based
//     tokens, or the static credentials. See AuthorizationHeader for the order.
//   - the tenant override of [WithTenant], which selects the tenant of Basic
//     credentials (client credentials only; credentials in ctx keep their own
//     tenant)
//   - User-Agent and X-APPLICATION, and the headers of [WithExtraHeaders]
//   - everything the client's request middleware ([Client.Use],
//     ClientOptions.Middleware) adds or changes on the request
//
// The returned header never includes the websocket handshake headers (Upgrade,
// Connection, Sec-WebSocket-*): the dialer owns those.
//
// Middleware: the upgrade request (GET on the base URL, with the Upgrade
// headers) is passed through the middleware chain, ending in a function which
// captures the final headers. Only what a middleware does to the request
// (headers, host) applies. The response side of a middleware sees a synthetic
// empty "101 Switching Protocols" response, so a middleware that needs the real
// response (retries, response logging) has no effect. Middleware that rejects
// the request (returns an error) makes WebsocketHeader fail.
func (c *Client) WebsocketHeader(ctx context.Context) (http.Header, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.clientMu.Lock()
	base := *c.BaseURL
	c.clientMu.Unlock()
	req, err := c.websocketRequest(ctx, base.String(), nil)
	if err != nil {
		return nil, err
	}
	return dialHeader(req), nil
}

// DialWebsocket opens the websocket at rawURL (ws, wss, http or https) with the
// settings of this client: [Client.WebsocketDialer] for TLS and proxy and
// [Client.WebsocketHeader] for the request headers. header adds request headers
// (for example a subprotocol); it is applied before the context's extra headers,
// and an Authorization header in it is kept instead of the client's.
//
// ctx cancels dialing and the handshake.
//
// Dry run: when [IsDryRun] is true for ctx nothing is dialed. The upgrade
// request is handed to the dry-run handler ([WithDryRunHandler]) or logged by
// slog ("DRY RUN"), exactly like an HTTP request, with Authorization redacted
// in the log, and a *[wsdial.DryRunError] is returned (errors.Is(err,
// wsdial.ErrDryRun)) carrying that request. A "token" query parameter of the URL
// (notification2) is redacted unless redaction is disabled.
//
// On success resp.Request is the upgrade request which was sent.
func (c *Client) DialWebsocket(ctx context.Context, rawURL string, header http.Header) (*websocket.Conn, *http.Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	dialURL, err := websocketURL(rawURL)
	if err != nil {
		return nil, nil, err
	}
	req, err := c.websocketRequest(ctx, dialURL, header)
	if err != nil {
		return nil, nil, err
	}

	if ctxhelpers.IsDryRun(ctx) {
		return nil, nil, c.reportWebsocketDryRun(ctx, req)
	}

	conn, resp, err := dialContext(ctx, c.WebsocketDialer(), dialURL, dialHeader(req))
	if err != nil {
		return nil, resp, err
	}
	if resp != nil {
		resp.Request = req
	}
	return conn, resp, nil
}

// dialContext is Dialer.DialContext which also stops waiting for the handshake
// response when ctx ends: gorilla only applies the deadline of ctx to the
// handshake, not its cancellation. The error then wraps ctx.Err().
func dialContext(ctx context.Context, dialer *websocket.Dialer, rawURL string, header http.Header) (*websocket.Conn, *http.Response, error) {
	var mu sync.Mutex
	var netConn net.Conn
	baseDial := dialer.NetDialContext
	if baseDial == nil {
		baseDial = (&net.Dialer{}).DialContext
	}
	dialer.NetDialContext = func(dctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := baseDial(dctx, network, addr)
		if err != nil {
			return nil, err
		}
		mu.Lock()
		netConn = conn
		mu.Unlock()
		if ctx.Err() != nil {
			_ = conn.Close()
		}
		return conn, nil
	}
	stop := context.AfterFunc(ctx, func() {
		mu.Lock()
		defer mu.Unlock()
		if netConn != nil {
			_ = netConn.Close()
		}
	})

	conn, resp, err := dialer.DialContext(ctx, rawURL, header)
	if !stop() && err == nil {
		// ctx ended while the connection was established: it may be closed
		_ = conn.Close()
		return nil, resp, fmt.Errorf("websocket dial: %w", ctx.Err())
	}
	if err != nil {
		// The handshake deadline (that of ctx) can fire before ctx reports it
		ctxErr := ctx.Err()
		if deadline, ok := ctx.Deadline(); ctxErr == nil && ok && !time.Now().Before(deadline) {
			ctxErr = context.DeadlineExceeded
		}
		if ctxErr != nil {
			err = fmt.Errorf("websocket dial: %w (%w)", ctxErr, err)
		}
	}
	return conn, resp, err
}

// websocketURL normalises a URL to the ws/wss scheme.
func websocketURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid websocket url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
	default:
		return "", fmt.Errorf("invalid websocket url %q: unsupported scheme %q", rawURL, u.Scheme)
	}
	return u.String(), nil
}

// websocketRequest builds the upgrade request for rawURL and runs it through the
// request middleware chain. The result has the final headers.
func (c *Client) websocketRequest(ctx context.Context, rawURL string, extra http.Header) (*http.Request, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid websocket url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "ws":
		u.Scheme = "http"
	case "wss":
		u.Scheme = "https"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	ua := c.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}
	req.Header.Set("User-Agent", ua)
	req.Header.Set("X-APPLICATION", "go-client")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Version", "13")
	for k, values := range extra {
		req.Header[http.CanonicalHeaderKey(k)] = append([]string(nil), values...)
	}
	for k, v := range ctxhelpers.GetExtraHeaders(ctx) {
		req.Header.Set(k, v)
	}

	if req.Header.Get("Authorization") == "" {
		authz, err := c.websocketAuthorization(ctx)
		if err != nil {
			return nil, err
		}
		if authz != "" {
			req.Header.Set("Authorization", authz)
		}
	}

	if hook := ctxhelpers.RequestHook(ctx); hook != nil {
		if err := hook(req); err != nil {
			return nil, fmt.Errorf("prepare request: %w", err)
		}
	}

	c.clientMu.Lock()
	middleware := append([]Middleware(nil), c.middleware...)
	c.clientMu.Unlock()
	if len(middleware) == 0 {
		return req, nil
	}

	final := req
	capture := RoundTripperFunc(func(r *http.Request) (*http.Response, error) {
		final = r
		return &http.Response{
			Status:     "101 Switching Protocols",
			StatusCode: http.StatusSwitchingProtocols,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     http.Header{},
			Body:       http.NoBody,
			Request:    r,
		}, nil
	})
	resp, err := chainMiddleware(capture, middleware).RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("websocket request middleware: %w", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	return final, nil
}

// websocketAuthorization is the Authorization header of a websocket connection:
// [Client.AuthorizationHeader], with the tenant of WithTenant applied to the
// client's Basic credentials.
func (c *Client) websocketAuthorization(ctx context.Context) (string, error) {
	authz, err := c.AuthorizationHeader(ctx)
	if err != nil || authz == "" {
		return authz, err
	}
	tenant, ok := ctxhelpers.TenantFromContext(ctx)
	if !ok || tenant == "" || !strings.HasPrefix(authz, "Basic ") {
		return authz, nil
	}
	if _, hasCtxAuth := authentication.AuthFromContext(ctx); hasCtxAuth {
		return authz, nil
	}
	c.clientMu.Lock()
	auth := c.Auth
	c.clientMu.Unlock()
	if auth.Token != "" || auth.Username == "" || auth.Password == "" {
		return authz, nil
	}
	return basicAuthorization(authentication.JoinTenantUser(tenant, auth.Username), auth.Password), nil
}

// dialHeader returns the headers to give to the dialer: the request headers
// without the ones the dialer sets itself.
func dialHeader(req *http.Request) http.Header {
	h := req.Header.Clone()
	for _, k := range []string{
		"Upgrade", "Connection",
		"Sec-Websocket-Key", "Sec-Websocket-Version", "Sec-Websocket-Extensions",
	} {
		h.Del(k)
	}
	if req.URL != nil && req.Host != "" && req.Host != req.URL.Host {
		h.Set("Host", req.Host)
	}
	return h
}

// reportWebsocketDryRun hands the upgrade request to the dry-run handler or
// logs it, and returns the error which stands in for the connection.
func (c *Client) reportWebsocketDryRun(ctx context.Context, req *http.Request) error {
	report := req.Clone(context.Background())
	redact := ctxhelpers.ShouldRedactHeaders(ctx)
	if redact && report.URL != nil {
		q := report.URL.Query()
		if q.Has("token") {
			q.Set("token", "[REDACTED]")
			report.URL.RawQuery = q.Encode()
		}
	}
	if handler := ctxhelpers.DryRunHandler(ctx); handler != nil {
		handler(report)
	} else {
		headers := report.Header
		if redact {
			headers = redactSensitiveHeaders(report.Header)
		}
		slog.Info("DRY RUN",
			"method", report.Method,
			"url", report.URL.String(),
			"headers", headers,
		)
	}
	return &wsdial.DryRunError{Request: report}
}
