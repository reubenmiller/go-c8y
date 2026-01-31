package op

import (
	"context"
	"errors"
	"time"
)

// RequestOption is a functional option for configuring requests
type RequestOption func(*RequestConfig)

// RequestConfig holds configuration for API requests
type RequestConfig struct {
	Headers     map[string]string
	QueryParams map[string][]string
	Timeout     time.Duration
	RetryConfig *RetryConfig
	Tenant      string
}

// NewRequestConfig creates a new request configuration from options
func NewRequestConfig(opts ...RequestOption) RequestConfig {
	cfg := RequestConfig{
		Headers:     make(map[string]string),
		QueryParams: make(map[string][]string),
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

// WithHeader adds a header to the request
func WithHeader(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		c.Headers[key] = value
	}
}

// WithHeaders adds multiple headers to the request
func WithHeaders(headers map[string]string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(map[string]string)
		}
		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithQuery adds a query parameter to the request
func WithQuery(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.QueryParams == nil {
			c.QueryParams = make(map[string][]string)
		}
		c.QueryParams[key] = append(c.QueryParams[key], value)
	}
}

// WithQueryParams adds multiple query parameters to the request
func WithQueryParams(params map[string][]string) RequestOption {
	return func(c *RequestConfig) {
		if c.QueryParams == nil {
			c.QueryParams = make(map[string][]string)
		}
		for k, values := range params {
			c.QueryParams[k] = append(c.QueryParams[k], values...)
		}
	}
}

// WithTimeout sets the request timeout
func WithTimeout(timeout time.Duration) RequestOption {
	return func(c *RequestConfig) {
		c.Timeout = timeout
	}
}

// WithRetry sets the retry configuration
func WithRetry(config RetryConfig) RequestOption {
	return func(c *RequestConfig) {
		c.RetryConfig = &config
	}
}

// WithTenant sets the tenant for the request
func WithTenant(tenant string) RequestOption {
	return func(c *RequestConfig) {
		c.Tenant = tenant
	}
}

// Context keys
type requestConfigKey struct{}

// WithRequestOptions stores request options in context
func WithRequestOptions(ctx context.Context, opts ...RequestOption) context.Context {
	cfg := NewRequestConfig(opts...)
	return context.WithValue(ctx, requestConfigKey{}, &cfg)
}

// RequestConfigFromContext retrieves request config from context
func RequestConfigFromContext(ctx context.Context) *RequestConfig {
	cfg, _ := ctx.Value(requestConfigKey{}).(*RequestConfig)
	return cfg
}

// IsRetryableError determines if an error can be retried
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// Network/timeout errors
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Canceled) {
		return false // Don't retry cancelled/timeout contexts
	}

	// Check for common retryable patterns
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"timeout",
		"503",
		"502",
		"500",
		"429", // Rate limit
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

// IsNotFoundError checks if an error is a 404 Not Found
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrNotFound) {
		return true
	}

	errStr := err.Error()
	return contains(errStr, "404") || contains(errStr, "not found")
}
