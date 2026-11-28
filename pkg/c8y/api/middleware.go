package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/mock"
	"resty.dev/v3"
)

// snapshotRequestBody reads req.Body into a new byte slice and replaces the
// body with a reader over that copy, so the request stays readable after the
// HTTP client has released its own body buffer. It returns nil for requests
// without a body.
func snapshotRequestBody(req *http.Request) ([]byte, error) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, nil
	}
	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	setRequestBody(req, body)
	return body, nil
}

// setRequestBody points req.Body and req.GetBody at body. It does nothing for
// requests without a body.
func setRequestBody(req *http.Request, body []byte) {
	if body == nil {
		return
	}
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(body)), nil
	}
	req.ContentLength = int64(len(body))
}

// StatsMap tracks counts by HTTP method and path (thread-safe)
type StatsMap struct {
	mu    sync.Mutex
	stats map[string]map[string]int64 // method -> path -> count
}

func NewStatsMap() *StatsMap {
	return &StatsMap{stats: make(map[string]map[string]int64)}
}

// Inc increments the count for the given method and path
func (s *StatsMap) Inc(method, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stats[method] == nil {
		s.stats[method] = make(map[string]int64)
	}
	s.stats[method][path]++
}

// Get returns the count for a method and path
func (s *StatsMap) Get(method, path string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stats[method][path]
}

// All returns a copy of the stats map
func (s *StatsMap) All() map[string]map[string]int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]map[string]int64, len(s.stats))
	for m, paths := range s.stats {
		out[m] = make(map[string]int64, len(paths))
		for p, c := range paths {
			out[m][p] = c
		}
	}
	return out
}

// DryRunTransport wraps an http.RoundTripper and intercepts requests when dry run is enabled
type DryRunTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()

	// Strip the Authorization header for requests that should be unauthenticated
	// (AuthTypeNone). This runs at the transport layer so it takes effect after
	// all resty middleware and addCredentials have had a chance to set it.
	// We modify req.Header in-place (rather than cloning) so that the debug
	// middleware, which reads resp.Request.RawRequest.Header, sees the accurate
	// outgoing headers.
	if ctxhelpers.IsNoAuth(ctx) {
		req.Header.Del("Authorization")
	} else if req.Header.Get("Authorization") == emptyBasicAuthorization {
		// SetAuth with no credentials resets the HTTP client's basic credentials
		// to empty values (resty cannot unset them); do not send them.
		req.Header.Del("Authorization")
	}
	// --noAccept: drop the Accept header so a POST/PUT returns the resource as
	// stored rather than a forced representation. Stripped here (the transport)
	// so it applies after all resty middleware has set it.
	if ctxhelpers.IsNoAccept(ctx) {
		req.Header.Del("Accept")
	}

	// RequestOptions.PrepareRequest: runs here, on the final *http.Request,
	// before the dry-run decision so dry-run output shows its changes.
	if hook := ctxhelpers.RequestHook(ctx); hook != nil {
		if err := hook(req); err != nil {
			return nil, fmt.Errorf("prepare request: %w", err)
		}
	}

	// Keep an SDK-owned copy of the request (including its body) for callers
	// that inspect it after the call, e.g. Result.Request.
	if capture := ctxhelpers.GetRequestCapture(ctx); capture != nil {
		body, err := snapshotRequestBody(req)
		if err != nil {
			return nil, err
		}
		captured := req.Clone(context.Background())
		setRequestBody(captured, body)
		if body == nil {
			captured.Body = nil
			captured.GetBody = nil
		}
		capture.Set(captured)
	}

	isDryRun := ctxhelpers.IsDryRun(ctx)
	isMockResponses := ctxhelpers.IsMockResponses(ctx)

	// If neither flag is set, proceed normally
	if !isDryRun && !isMockResponses {
		// WithRequestApprover: ask before sending a request that changes data.
		// This runs on the final request (ids resolved, every option applied)
		// and only for real sends: dry runs, mocks and deferred preparation
		// never reach it. A declined request is not sent; its body is closed
		// so a streaming (io.Pipe) body writer is released.
		if err := ctxhelpers.ApproveRequest(ctx, req); err != nil {
			if req.Body != nil {
				_ = req.Body.Close()
			}
			return nil, err
		}
		resp, err := t.Transport.RoundTrip(req)
		if err == nil && resp != nil && resp.Body != nil {
			if wrap := ctxhelpers.ResponseBodyWrapper(ctx); wrap != nil {
				if body := wrap(resp); body != nil {
					resp.Body = body
				}
			}
		}
		return resp, err
	}

	// The request is not sent, but callers inspect it afterwards through
	// op.Result.Request. Copy the body now: the HTTP client may reuse the
	// buffer behind req.Body once this round trip returns.
	reqBody, err := snapshotRequestBody(req)
	if err != nil {
		return nil, err
	}

	// If dry run is enabled, hand the request to a registered handler (e.g. a
	// CLI renderer), falling back to a slog line.
	if isDryRun {
		if handler := ctxhelpers.DryRunHandler(ctx); handler != nil {
			handler(req)
			// The handler may have read the body; rewind it for later inspection.
			setRequestBody(req, reqBody)
		} else {
			headers := req.Header
			if ctxhelpers.ShouldRedactHeaders(ctx) {
				headers = redactSensitiveHeaders(req.Header)
			}
			slog.Info("DRY RUN",
				"method", req.Method,
				"url", req.URL.String(),
				"headers", headers,
			)
		}
	}

	// If mock responses are enabled, return mock data
	if isMockResponses {
		// Create a mock response based on the request method
		statusCode := http.StatusOK
		var body []byte
		var err error

		switch req.Method {
		case http.MethodPost:
			statusCode = http.StatusCreated
			// Detect response type from URL and return single item
			responseType := mock.DetectResponseType(req.URL.Path, false)
			body, err = mock.GetResponse(responseType)
			if err != nil {
				body = []byte(`{"id":"dry-run-id","self":"https://dry-run.example.com","creationTime":"2024-01-01T00:00:00.000Z"}`)
			}
		case http.MethodDelete:
			statusCode = http.StatusNoContent
			body = []byte("")
		case http.MethodGet:
			// Determine if this is a collection or single item request
			// Simple heuristic: if path has specific ID patterns or ends with a known ID format
			isCollection := !hasResourceID(req.URL.Path)
			responseType := mock.DetectResponseType(req.URL.Path, isCollection)
			body, err = mock.GetResponse(responseType)
			if err != nil {
				if isCollection {
					body = []byte(`{"managedObjects":[],"statistics":{"currentPage":1,"pageSize":5}}`)
				} else {
					body = []byte(`{"id":"dry-run-id","name":"Dry Run Item","type":"dry-run","self":"https://dry-run.example.com"}`)
				}
			}
		case http.MethodPut:
			// Return single item response
			responseType := mock.DetectResponseType(req.URL.Path, false)
			body, err = mock.GetResponse(responseType)
			if err != nil {
				body = []byte(`{"id":"dry-run-id","self":"https://dry-run.example.com","lastUpdated":"2024-01-01T00:00:00.000Z"}`)
			}
		default:
			body = []byte(`{"message":"Dry run mode - request not sent"}`)
		}

		resp := &http.Response{
			Status:     http.StatusText(statusCode),
			StatusCode: statusCode,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewBuffer(body)),
			Request:    req,
		}

		// Set common response headers
		resp.Header.Set("Content-Type", "application/json")
		if isDryRun {
			resp.Header.Set("X-Dry-Run", "true")
		}
		if isMockResponses {
			resp.Header.Set("X-Mock-Response", "true")
		}

		return resp, nil
	}

	// Dry run without mock responses: never send the request. Return an empty
	// success response so callers receive no data rather than fabricated data.
	resp := &http.Response{
		Status:     http.StatusText(http.StatusNoContent),
		StatusCode: http.StatusNoContent,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(nil)),
		Request:    req,
	}
	resp.Header.Set("X-Dry-Run", "true")
	return resp, nil
}

// BaseTransport returns the underlying transport.
func (t *DryRunTransport) BaseTransport() http.RoundTripper {
	return t.Transport
}

// TLSClientConfig returns the TLS configuration from the underlying transport if available.
func (t *DryRunTransport) TLSClientConfig() *tls.Config {
	if t.Transport == nil {
		return nil
	}

	switch transport := t.Transport.(type) {
	case *http.Transport:
		return transport.TLSClientConfig
	case interface{ TLSClientConfig() *tls.Config }:
		return transport.TLSClientConfig()
	}

	return nil
}

// SetTLSClientConfig sets the TLS configuration on the underlying transport if possible.
func (t *DryRunTransport) SetTLSClientConfig(config *tls.Config) error {
	if t.Transport == nil {
		return fmt.Errorf("no underlying transport")
	}

	switch transport := t.Transport.(type) {
	case *http.Transport:
		transport.TLSClientConfig = config
		return nil
	case interface{ SetTLSClientConfig(*tls.Config) error }:
		return transport.SetTLSClientConfig(config)
	}

	return fmt.Errorf("underlying transport does not support TLS configuration")
}

// hasResourceID checks if the URL path appears to reference a specific resource ID
// Simple heuristic: paths ending with numeric or UUID-like patterns are single resources
func hasResourceID(path string) bool {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return false
	}
	lastPart := parts[len(parts)-1]

	// Check if last part looks like an ID (numeric or contains hyphens like UUIDs)
	if len(lastPart) > 0 {
		// Numeric ID or contains dashes (UUID-like)
		if strings.ContainsAny(lastPart, "0123456789") && !strings.Contains(lastPart, "?") {
			return true
		}
	}
	return false
}

// middlewareAddUserAgent sets the User-Agent and X-Application headers which
// identify the client. A header the caller already set (e.g. with
// WithExtraHeaders or in the request options) is kept.
func middlewareAddUserAgent(application string, userAgent string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		if r.Header.Get("User-Agent") == "" {
			r.SetHeader("User-Agent", userAgent)
		}
		if r.Header.Get("X-Application") == "" {
			r.SetHeader("X-APPLICATION", application)
		}
		return nil
	}
}

func middlewareAddHost(domain string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		if domain != "" && r.RawRequest != nil && domain != r.RawRequest.URL.Host {
			r.RawRequest.Host = domain
		}
		return nil
	}
}

var HeaderAuthorization = "Authorization"

func basicAuthorization(user, password string) string {
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(user+":"+password))
}

func bearerAuthorization(token string) string {
	return "Bearer " + token
}

// AuthorizationHeader returns the Authorization header value the client would
// send for a request made with ctx, or "" (and no error) when the client has no
// credentials for it. Use it for requests which are forwarded without going
// through the client, e.g. by a reverse proxy.
//
// The same precedence as a normal request applies:
//
//  1. Requests marked as unauthenticated (contexthelpers.WithNoAuth) return "".
//  2. Per-request credentials in ctx (api.WithAuth, api.WithServiceUser): a token
//     is sent as Bearer; username/password are sent as Basic, or exchanged for a
//     cached bearer token when context token exchange is enabled
//     (see SetContextTokenExchange), falling back to Basic.
//  3. The client's token source (credential-backed automatic source, device
//     certificate login, or an explicit AuthOptions.TokenSource). The token is
//     cached and renewed by the source when it expires, so a long-lived caller
//     can call AuthorizationHeader for each forwarded request. An error from the
//     source is returned.
//  4. The client's static credentials (Client.Auth) as SetAuth gives them to the
//     HTTP client: the token when one is set (it overrides Basic on the wire),
//     otherwise Basic from username and password.
//
// The client's AuthType (AuthOptions.AuthType) applies as it does for a request:
// Basic returns the Basic header of the username and password (no token), None
// returns "".
//
// Renewal is limited to what the token source does (renew on expiry). The
// client's retry-on-401 logic only runs for requests the client sends itself, so
// a caller forwarding requests must call Login again if the server rejects the
// token before it expires.
func (c *Client) AuthorizationHeader(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if ctxhelpers.IsNoAuth(ctx) {
		return "", nil
	}
	if auth, ok := authentication.AuthFromContext(ctx); ok {
		if v := c.contextAuthorization(ctx, auth); v != "" {
			return v, nil
		}
	} else if !ctxhelpers.IsExplicitAuthType(ctx) {
		switch c.authType() {
		case authentication.AuthTypeNone:
			return "", nil
		case authentication.AuthTypeBasic:
			c.clientMu.Lock()
			auth := c.Auth
			c.clientMu.Unlock()
			return basicStaticAuthorization(auth), nil
		}
	}
	if _, ok := authentication.AuthFromContext(ctx); !ok {
		if v, err := c.tokenSourceAuthorization(ctx); err != nil || v != "" {
			return v, err
		}
	}
	c.clientMu.Lock()
	auth := c.Auth
	c.clientMu.Unlock()
	return staticAuthorization(auth), nil
}

// basicStaticAuthorization is the Basic header for the client's credentials.
func basicStaticAuthorization(auth authentication.AuthOptions) string {
	if auth.Username != "" && auth.Password != "" {
		return basicAuthorization(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
	}
	return ""
}

// staticAuthorization is the header the HTTP client sends for credentials set
// with SetAuth: resty sets Basic first and the token afterwards, so a token wins.
func staticAuthorization(auth authentication.AuthOptions) string {
	if auth.Token != "" {
		return bearerAuthorization(auth.Token)
	}
	if auth.Username != "" && auth.Password != "" {
		return basicAuthorization(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
	}
	return ""
}

// contextAuthorization mirrors middlewareContextAuthorization for credentials
// carried in the context.
func (c *Client) contextAuthorization(ctx context.Context, auth authentication.AuthOptions) string {
	authType := effectiveAuthType(auth)
	if authType == authentication.AuthTypeNone {
		return ""
	}
	hasBasic := auth.Username != "" && auth.Password != ""
	if auth.Token != "" && (authType != authentication.AuthTypeBasic || !hasBasic) {
		return bearerAuthorization(auth.Token)
	}
	if !hasBasic {
		return ""
	}
	if authType != authentication.AuthTypeBasic && !ctxhelpers.IsSkipContextTokenExchange(ctx) {
		if token := c.contextTokenFor(auth); token != "" {
			return bearerAuthorization(token)
		}
	}
	return basicAuthorization(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
}

// tokenSourceAuthorization mirrors tokenSourceMiddleware.
func (c *Client) tokenSourceAuthorization(ctx context.Context) (string, error) {
	source := c.currentTokenSource()
	if source == nil || ctxhelpers.IsSkipTokenSource(ctx) || ctxhelpers.IsDryRun(ctx) {
		return "", nil
	}
	tok, err := source.Token()
	if err != nil {
		return "", fmt.Errorf("token source: %w", err)
	}
	if tok == nil || tok.AccessToken == "" {
		return "", nil
	}
	return bearerAuthorization(tok.AccessToken), nil
}

func middlewareRemoveEmptyTenantID() resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		// Set tenant id based on the context
		if currentValue, ok := r.PathParams[core.PathParamTenantID]; ok && currentValue == "" {
			// remove any empty values in the request so the client setting
			// takes priority
			delete(r.PathParams, core.PathParamTenantID)
		}

		// Allow overriding using context (see WithTenant)
		if tenant, ok := ctxhelpers.TenantFromContext(r.Context()); ok && tenant != "" {
			r.SetPathParam(core.PathParamTenantID, tenant)
		}
		return nil
	}
}

// middlewareContextAuthorization applies per-request credentials carried in the
// request context (see WithAuth / authentication.WithAuth). When credentials are
// present in the context they take priority over the client-level credentials
// and token source, allowing a single shared client to act on behalf of multiple
// tenants (e.g. service users inside a MULTI_TENANT microservice).
//
// When the client has context token exchange enabled (see
// Client.SetContextTokenExchange), basic credentials are exchanged for a cached
// OAI-Secure bearer token, falling back to basic auth when the exchange is not
// possible.
func middlewareContextAuthorization(c *Client) resty.RequestMiddleware {
	return func(_ *resty.Client, r *resty.Request) error {
		auth, ok := authentication.AuthFromContext(r.Context())
		if !ok {
			return nil
		}
		// Clear any client-level token so the per-request credentials win
		r.SetAuthToken("")
		authType := effectiveAuthType(auth)
		if authType == authentication.AuthTypeNone {
			r.SetContext(ctxhelpers.WithNoAuth(r.Context()))
			return nil
		}
		hasBasic := auth.Username != "" && auth.Password != ""
		switch {
		case auth.Token != "" && (authType != authentication.AuthTypeBasic || !hasBasic):
			r.SetAuthScheme("Bearer")
			r.SetAuthToken(auth.Token)
		case hasBasic:
			token := ""
			if authType != authentication.AuthTypeBasic && !ctxhelpers.IsSkipContextTokenExchange(r.Context()) {
				token = c.contextTokenFor(auth)
			}
			if token != "" {
				r.SetAuthScheme("Bearer")
				r.SetAuthToken(token)
			} else {
				r.SetAuthScheme("Basic")
				r.SetBasicAuth(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
			}
		}
		// Also update the {tenantId} path parameter used in tenant-scoped URLs
		if auth.Tenant != "" {
			r.SetPathParam(core.PathParamTenantID, auth.Tenant)
		}
		return nil
	}
}

// emptyBasicAuthorization is the header resty builds from an empty username and
// password, which is how setAuth clears the HTTP client's basic credentials.
const emptyBasicAuthorization = "Basic Og=="

// setAuth makes the HTTP client send exactly the credentials in auth: values the
// options leave empty are cleared, so a token or basic credentials of an earlier
// call are not sent any more.
func setAuth(c *resty.Client, auth authentication.AuthOptions) {
	if auth.CertificateKey != "" && auth.Certificate != "" {
		if _, err := os.Stat(auth.CertificateKey); err == nil {
			c.SetCertificateFromFile(auth.Certificate, auth.CertificateKey)
		} else {
			c.SetCertificateFromString(auth.Certificate, auth.CertificateKey)
		}
	}
	c.SetAuthToken(auth.Token)
	if auth.Username != "" && auth.Password != "" {
		c.SetBasicAuth(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
	} else {
		c.SetBasicAuth("", "")
	}
}

// effectiveAuthType is the authentication type a request without its own type
// uses: the first entry of auth.AuthType which is not AuthTypeUnset, otherwise
// AuthTypeUnset (use whatever credentials are present).
func effectiveAuthType(auth authentication.AuthOptions) authentication.AuthType {
	for _, t := range auth.AuthType {
		if t != authentication.AuthTypeUnset {
			return t
		}
	}
	return authentication.AuthTypeUnset
}

// middlewareAuthType applies the client's AuthType to requests which did not
// choose a type themselves (see TryRequest.WithAuthType). It must run before the
// token source middleware: Basic and None skip the token source and drop the
// bearer token so that only basic credentials (or none) are sent, None also
// strips the header at the transport. Requests with per-request credentials
// (WithAuth) are left alone.
func middlewareAuthType(c *Client) resty.RequestMiddleware {
	return func(_ *resty.Client, r *resty.Request) error {
		ctx := r.Context()
		if ctxhelpers.IsExplicitAuthType(ctx) {
			return nil
		}
		if _, ok := authentication.AuthFromContext(ctx); ok {
			return nil
		}
		switch c.authType() {
		case authentication.AuthTypeBasic:
			r.SetAuthToken("")
			r.SetContext(ctxhelpers.WithSkipTokenSource(ctx))
		case authentication.AuthTypeNone:
			r.SetAuthToken("")
			r.SetContext(ctxhelpers.WithNoAuth(ctxhelpers.WithSkipTokenSource(ctx)))
		}
		return nil
	}
}

// tokenSourceMiddleware returns a resty.RequestMiddleware that injects a bearer token
// from the TokenSource returned by getSource before every outgoing request.
//
// Requests carrying a WithSkipTokenSource context value are left untouched so that
// internal credential-fetch calls (e.g. username/password login) do not
// recursively trigger token renewal.
func tokenSourceMiddleware(getSource func() authentication.TokenSource) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		src := getSource()
		if src == nil {
			return nil
		}
		if ctxhelpers.IsSkipTokenSource(req.Context()) {
			return nil
		}
		// Per-request credentials (see WithAuth) take priority over the client's
		// token source, so skip token acquisition entirely for these requests.
		if _, ok := authentication.AuthFromContext(req.Context()); ok {
			return nil
		}
		// Skip token acquisition for dry-run requests: the DryRunTransport intercepts
		// the request at the HTTP layer and never sends it to the real server, so
		// acquiring (and potentially refreshing) a token is unnecessary and would
		// break tests that run without live credentials.
		if ctxhelpers.IsDryRun(req.Context()) {
			return nil
		}
		tok, err := src.Token()
		if err != nil {
			return fmt.Errorf("token source: %w", err)
		}
		if tok != nil && tok.AccessToken != "" {
			req.SetAuthToken(tok.AccessToken)
		}
		return nil
	}
}

// redactSensitiveHeaders creates a copy of headers with sensitive values redacted
func redactSensitiveHeaders(headers http.Header) http.Header {
	// List of headers that should be redacted for security
	sensitiveHeaders := map[string]bool{
		"authorization":       true,
		"cookie":              true,
		"set-cookie":          true,
		"x-xsrf-token":        true,
		"x-csrf-token":        true,
		"api-key":             true,
		"x-api-key":           true,
		"apikey":              true,
		"x-auth-token":        true,
		"x-authorization":     true,
		"proxy-authorization": true,
	}

	redacted := make(http.Header, len(headers))
	for key, values := range headers {
		lowerKey := strings.ToLower(key)
		if sensitiveHeaders[lowerKey] {
			// Redact the value but keep the header name visible
			redacted[key] = []string{"[REDACTED]"}
		} else {
			// Copy non-sensitive headers as-is
			redacted[key] = values
		}
	}
	return redacted
}
