package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/mock"
	"resty.dev/v3"
)

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

// MiddlewareCountByMethodAndPath returns a resty.ResponseMiddleware that increments stats by HTTP method and path.
// Pass a pointer to a StatsMap to collect stats during execution.
// Uses ResponseMiddleware to capture the actual URL after path parameters are substituted.
// Note: Does not count dry run or mock responses, as they are not actually sent to the server.
func MiddlewareCountByMethodAndPath(stats *StatsMap) resty.ResponseMiddleware {
	return func(_ *resty.Client, r *resty.Response) error {
		// Skip counting if this was a dry run or mock response (not actually sent to server)
		if r.Request != nil && r.Request.RawRequest != nil {
			ctx := r.Request.RawRequest.Context()
			if ctxhelpers.IsDryRun(ctx) || ctxhelpers.IsMockResponses(ctx) {
				return nil
			}
		}

		path := ""
		method := ""

		// Get the actual request that was sent
		if r.Request != nil && r.Request.RawRequest != nil && r.Request.RawRequest.URL != nil {
			path = r.Request.RawRequest.URL.Path
			method = r.Request.Method
		}

		if path != "" && method != "" {
			stats.Inc(method, path)
		}
		return nil
	}
}

// MiddlewareCountRequests returns a resty.RequestMiddleware that increments the given counter for each HTTP request sent.
// Useful for gathering API call statistics in tests, benchmarks, or debugging.
func MiddlewareCountRequests(counter *int64) resty.RequestMiddleware {
	return func(_ *resty.Client, _ *resty.Request) error {
		atomic.AddInt64(counter, 1)
		return nil
	}
}

// WithDryRun returns a context with dry run enabled
// Dry run mode logs requests for inspection/validation without sending them
func WithDryRun(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithDryRun(ctx, enabled)
}

// IsDryRun checks if dry run is enabled in the context
func IsDryRun(ctx context.Context) bool {
	return ctxhelpers.IsDryRun(ctx)
}

// WithMockResponses returns a context with mock responses enabled
// When enabled, HTTP requests will return mock data from embedded JSON files
// instead of making real API calls. Useful for unit testing without network dependencies.
// Can be combined with WithDryRun for logging + mock data, or used independently.
func WithMockResponses(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithMockResponses(ctx, enabled)
}

// IsMockResponses checks if mock responses are enabled in the context
func IsMockResponses(ctx context.Context) bool {
	return ctxhelpers.IsMockResponses(ctx)
}

// WithRedactHeaders returns a context with header redaction control
// By default, headers are redacted for security. Set to false to disable redaction for debugging.
// Example: ctx = api.WithRedactHeaders(ctx, false) // Disable redaction for debugging
func WithRedactHeaders(ctx context.Context, redact bool) context.Context {
	return ctxhelpers.WithRedactHeaders(ctx, redact)
}

// ShouldRedactHeaders checks if header redaction is enabled in the context
// Returns true by default (secure by default)
func ShouldRedactHeaders(ctx context.Context) bool {
	return ctxhelpers.ShouldRedactHeaders(ctx)
}

// WithDeferredExecution returns a context with deferred execution enabled
// When enabled, operations will prepare the request (including parameter resolution)
// but won't execute the HTTP call until Result.Execute() is called.
// This is useful for confirmation prompts before destructive operations.
func WithDeferredExecution(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithDeferredExecution(ctx, enabled)
}

// IsDeferredExecution checks if deferred execution is enabled in the context
func IsDeferredExecution(ctx context.Context) bool {
	return ctxhelpers.IsDeferredExecution(ctx)
}

// DryRunTransport wraps an http.RoundTripper and intercepts requests when dry run is enabled
type DryRunTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	isDryRun := ctxhelpers.IsDryRun(ctx)
	isMockResponses := ctxhelpers.IsMockResponses(ctx)

	// If neither flag is set, proceed normally
	if !isDryRun && !isMockResponses {
		return t.Transport.RoundTrip(req)
	}

	// If dry run is enabled, log the request
	if isDryRun {
		// Determine which headers to log (redacted or full)
		headers := req.Header
		if ctxhelpers.ShouldRedactHeaders(ctx) {
			headers = redactSensitiveHeaders(req.Header)
		}

		// Log the request details
		slog.Info("DRY RUN",
			"method", req.Method,
			"url", req.URL.String(),
			"headers", headers,
		)
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

	// Not a dry run, proceed with the actual request
	return t.Transport.RoundTrip(req)
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

func MiddlewareAddUserAgent(application string, userAgent string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		r.SetHeader("User-Agent", userAgent)
		r.SetHeader("X-APPLICATION", application)
		return nil
	}
}

func MiddlewareAddHost(domain string) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		if domain != "" && r.RawRequest != nil && domain != r.RawRequest.URL.Host {
			r.RawRequest.Host = domain
		}
		return nil
	}
}

func MiddlewareAddCookies(cookies []*http.Cookie) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		for _, cookie := range cookies {
			if cookie.Name == "XSRF-TOKEN" {
				r.SetHeader("X-"+cookie.Name, cookie.Value)
			} else {
				r.SetCookie(cookie)
			}
		}
		return nil
	}
}

var HeaderAuthorization = "Authorization"

func MiddlewareAuthorization(auth authentication.AuthOptions) resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		// Don't override the authorization header if already set
		if v := r.Header.Get(HeaderAuthorization); v != "" {
			return nil
		}
		for _, authType := range auth.GetAuthTypes() {
			switch authType {
			case authentication.AuthTypeBasic:
				user := authentication.JoinTenantUser(auth.Tenant, auth.Username)
				if user != "" && auth.Password != "" {
					r.SetBasicAuth(user, auth.Password)
					return nil
				}
			case authentication.AuthTypeBearer:
				if auth.Token != "" {
					r.Header.Set(HeaderAuthorization, fmt.Sprintf("Bearer %s", auth.Token))
					slog.Info("Auth", "value", r.Header.Get(HeaderAuthorization))

					return nil
				}
			case authentication.AuthTypeUnset:
				if auth.Token != "" {
					r.Header.Set(HeaderAuthorization, fmt.Sprintf("Bearer %s", auth.Token))
					slog.Info("Auth", "value", r.Header.Get(HeaderAuthorization))
					return nil
				}
				user := authentication.JoinTenantUser(auth.Tenant, auth.Username)
				if user != "" && auth.Password != "" {
					r.SetBasicAuth(user, auth.Password)
					return nil
				}
			case authentication.AuthTypeNone:
				return nil
			}
		}

		return nil
	}
}

func MiddlewareRemoveEmptyTenantID() resty.RequestMiddleware {
	return func(c *resty.Client, r *resty.Request) error {
		// Set tenant id based on the context
		if currentValue, ok := r.PathParams["tenantID"]; ok && currentValue == "" {
			// remove any empty values in the request so the client setting
			// takes priority
			delete(r.PathParams, "tenantID")
		}

		// Allow overriding using context
		switch v := r.Context().Value("tenant").(type) {
		case string:
			if v != "" {
				r.SetPathParam("tenantID", v)
			}
		}
		return nil
	}
}

func SetAuth(c *resty.Client, auth authentication.AuthOptions) {
	if auth.CertificateKey != "" && auth.Certificate != "" {
		if _, err := os.Stat(auth.CertificateKey); err == nil {
			c.SetCertificateFromFile(auth.Certificate, auth.CertificateKey)
		} else {
			c.SetCertificateFromString(auth.Certificate, auth.CertificateKey)
		}
	}
	if auth.Token != "" {
		c.SetAuthToken(auth.Token)
	}
	if auth.Username != "" && auth.Password != "" {
		c.SetBasicAuth(authentication.JoinTenantUser(auth.Tenant, auth.Username), auth.Password)
	}
}

// TokenSourceMiddleware returns a resty.RequestMiddleware that injects a bearer token
// from the TokenSource returned by getSource before every outgoing request.
//
// Requests carrying a WithSkipTokenSource context value are left untouched so that
// internal credential-fetch calls (e.g. username/password login) do not
// recursively trigger token renewal.
func TokenSourceMiddleware(getSource func() authentication.TokenSource) resty.RequestMiddleware {
	return func(_ *resty.Client, req *resty.Request) error {
		src := getSource()
		if src == nil {
			return nil
		}
		if ctxhelpers.IsSkipTokenSource(req.Context()) {
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
