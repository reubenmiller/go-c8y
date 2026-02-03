package c8y_api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/mock"
	"resty.dev/v3"
)

// WithDryRun returns a context with dry run enabled
func WithDryRun(ctx context.Context, enabled bool) context.Context {
	return ctxhelpers.WithDryRun(ctx, enabled)
}

// IsDryRun checks if dry run is enabled in the context
func IsDryRun(ctx context.Context) bool {
	return ctxhelpers.IsDryRun(ctx)
}

// WithRedactHeaders returns a context with header redaction control
// By default, headers are redacted for security. Set to false to disable redaction for debugging.
// Example: ctx = c8y_api.WithRedactHeaders(ctx, false) // Disable redaction for debugging
func WithRedactHeaders(ctx context.Context, redact bool) context.Context {
	return ctxhelpers.WithRedactHeaders(ctx, redact)
}

// ShouldRedactHeaders checks if header redaction is enabled in the context
// Returns true by default (secure by default)
func ShouldRedactHeaders(ctx context.Context) bool {
	return ctxhelpers.ShouldRedactHeaders(ctx)
}

// DryRunTransport wraps an http.RoundTripper and intercepts requests when dry run is enabled
type DryRunTransport struct {
	Transport http.RoundTripper
}

// RoundTrip implements http.RoundTripper
func (t *DryRunTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if ctxhelpers.IsDryRun(req.Context()) {
		// Determine which headers to log (redacted or full)
		headers := req.Header
		if ctxhelpers.ShouldRedactHeaders(req.Context()) {
			headers = redactSensitiveHeaders(req.Header)
		}

		// Log the request details
		slog.Info("DRY RUN",
			"method", req.Method,
			"url", req.URL.String(),
			"headers", headers,
		)

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
		resp.Header.Set("X-Dry-Run", "true")

		return resp, nil
	}

	// Not a dry run, proceed with the actual request
	return t.Transport.RoundTrip(req)
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
			// setting the Host header actually does nothing however
			// it makes the setting visible when logging
			r.Header.Set("Host", domain)
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
