package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/applications"
	appversions "github.com/reubenmiller/go-c8y/pkg/c8y/api/applications/versions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/auditrecords"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/bulkoperations"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/events"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/features"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/loginoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/microservices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/notification2"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/operations"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/realtime"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/remoteaccess"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/repository"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/retentionrules"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/applicationplugins"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/plugins"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/plugins/versions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/zalando/go-keyring"
	"resty.dev/v3"
)

var ErrNotFound = errors.New("item: not found")

type AuthFunc func(r *http.Request) (string, error)

// DefaultRequestOptions default request options which are added to each outgoing request
type DefaultRequestOptions struct {
	DryRun bool

	// DryRunResponse return a mock response when using dry run
	DryRunResponse bool

	// DryRunHandler called when a request should be called
	// DryRunHandler func(options *RequestOptions, req *http.Request)
}

type Service struct {
	Client *resty.Client
}

// A Client manages communication with the Cumulocity API.
type Client struct {
	clientMu sync.Mutex    // clientMu protects the client during calls that modify the CheckRedirect func.
	Client   *resty.Client // HTTP client used to communicate with the API.

	RealtimeClient *realtime.Client

	// Show sensitive information
	showSensitive bool

	// Base URL for API requests. Defaults to the public Cumulocity API, but can be
	// set to a domain endpoint to use with Cumulocity. BaseURL should
	// always be specified with a trailing slash.
	BaseURL *url.URL

	// Domain. This can be different to the BaseURL when using a proxy or a custom alias
	Domain string

	// User agent used when communicating with the Cumulocity API.
	UserAgent string

	Auth authentication.AuthOptions

	// tokenSource is the active bearer-token provider. It is set automatically
	// from Auth credentials in NewClient, or can be supplied via AuthOptions.TokenSource.
	tokenSource authentication.TokenSource

	UseKeyRing bool

	// Cumulocity Version
	Version string

	UseTenantInUsername bool

	common core.Service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the Cumulocity API.
	// Context              *ContextService
	Alarms         *alarms.Service
	AuditRecords   *auditrecords.Service
	BulkOperations *bulkoperations.Service

	Measurements *measurements.Service
	Binaries     *binaries.Service

	LoginOptions *loginoptions.Service

	LoginTokens *logintokens.Service

	Devices              *devices.Service
	ManagedObjects       *managedobjects.Service
	Operations           *operations.Service
	Tenants              *tenants.Service
	Events               *events.Service
	Applications         *applications.Service
	ApplicationVersions  *appversions.Service
	Microservices        *microservices.Service
	Repository           *repository.Service
	UIPlugins            *plugins.Service
	UIPluginVersions     *versions.Service
	UIApplicationPlugins *applicationplugins.Service
	Identity             *identity.Service
	TrustedCertificates  *trustedcertificates.Service
	Notification2        *notification2.Service
	RemoteAccess         *remoteaccess.Service
	RetentionRules       *retentionrules.Service
	Users                *users.Service
	UserGroups           *usergroups.Service
	UserRoles            *userroles.Service
	Features             *features.Service
}

const (
	defaultUserAgent = "go-client-v2"
)

var (
	// EnvVarLoggerHideSensitive environment variable name used to control whether sensitive session information is logged or not. When set to "true", then the tenant, username, password, base 64 passwords will be obfuscated from the log messages
	EnvVarLoggerHideSensitive = "C8Y_LOGGER_HIDE_SENSITIVE"
)

// Format the base url to ensure it is normalized for cases where the
// scheme can be missing, and the trailing slash (which affects the default path used in calls)
func FormatBaseURL(v string) string {
	// add a default scheme if missing
	if !strings.Contains(v, "://") {
		v = "https://" + v
	}

	if strings.HasSuffix(v, "/") {
		return v
	}
	return v + "/"
}

// Client options
type ClientOptions struct {
	BaseURL string

	Auth authentication.AuthOptions

	// Create a realtime client
	Realtime bool

	// Show sensitive information in the logs
	ShowSensitive bool

	// Enable Keyring for saving and retrieving a token
	UseKeyRing bool

	// Not recommended
	InsecureSkipVerify bool

	Agent string

	Transport http.RoundTripper
}

func (c *Client) Realtime() *realtime.Client {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	if c.common.RealtimeClient == nil {
		c.common.RealtimeClient = realtime.NewClient(nil, realtime.ClientOptions{
			Host:     c.BaseURL.String(),
			Tenant:   c.Auth.Tenant,
			Username: c.Auth.Username,
			Password: c.Auth.Password,
			Token:    c.Auth.Token,
		})
	}
	return c.common.RealtimeClient
}

func NewClientFromEnvironment(opt ClientOptions) *Client {
	opt.BaseURL = authentication.HostFromEnvironment()
	opt.Auth = authentication.FromEnvironment()
	return NewClient(opt)
}

func SetCertificateChainHeaderIfRequired(client *resty.Client, auth authentication.AuthOptions) *resty.Client {
	if auth.Certificate == "" || auth.CertificateKey == "" {
		return client
	}

	certs := make([]tls.Certificate, 0)
	if _, err := os.Stat(auth.CertificateKey); err == nil {
		client.SetCertificateFromFile(auth.Certificate, auth.CertificateKey)
		if cert, err := tls.LoadX509KeyPair(auth.Certificate, auth.CertificateKey); err == nil {
			certs = append(certs, cert)
		}
	} else {
		client.SetCertificateFromString(auth.Certificate, auth.CertificateKey)
		if cert, err := tls.X509KeyPair([]byte(auth.Certificate), []byte(auth.CertificateKey)); err == nil {
			certs = append(certs, cert)
		}
	}
	if headerValue, err := certutil.CertificateChain(certs).Header(); err == nil && len(headerValue) > 0 {
		client.SetHeader(types.HeaderSSLCertificateChain, string(headerValue))
	}
	return client
}

// setTLSConfig attempts to set TLS configuration on a transport.
// It tries multiple approaches to handle different transport implementations.
func setTLSConfig(transport http.RoundTripper, insecureSkipVerify bool) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: insecureSkipVerify,
	}

	switch t := transport.(type) {
	case *http.Transport:
		t.TLSClientConfig = tlsConfig
		return nil
	case interface{ SetTLSClientConfig(*tls.Config) error }:
		return t.SetTLSClientConfig(tlsConfig)
	case interface{ BaseTransport() http.RoundTripper }:
		// Transport wrapper that exposes its base transport
		return setTLSConfig(t.BaseTransport(), insecureSkipVerify)
	}

	return fmt.Errorf("transport does not support TLS configuration")
}

// NewClient returns a new Cumulocity API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(opts ClientOptions) *Client {
	circuitBreaker := resty.NewCircuitBreakerWithCount(10, 5, 15*time.Second)

	rclient := resty.New().
		SetRetryCount(3).
		SetRetryWaitTime(2 * time.Second).
		SetRetryMaxWaitTime(30 * time.Second).
		SetResultError(core.APIError{}).
		SetCircuitBreaker(circuitBreaker)

	// Set any certificate before any other Transports are set as these will
	// make the TLS config inaccessible
	// TODO: Check if there is a better way to do this
	SetCertificateChainHeaderIfRequired(rclient, opts.Auth)

	targetBaseURL, _ := url.Parse(FormatBaseURL(opts.BaseURL))
	if targetBaseURL != nil {
		rclient.SetBaseURL(targetBaseURL.String())
	}

	// Configure base transport with TLS settings
	var baseTransport http.RoundTripper
	if opts.Transport != nil {
		// User provided a custom transport, use it as the base
		baseTransport = opts.Transport

		// Try to set TLS config on the base transport if it supports it
		if err := setTLSConfig(baseTransport, opts.InsecureSkipVerify); err != nil {
			slog.Debug("Could not set TLS config on custom transport", "err", err)
		}
	} else {
		// No custom transport, create a default one with TLS config
		defaultTransport := http.DefaultTransport.(*http.Transport).Clone()
		defaultTransport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
		}
		baseTransport = defaultTransport
	}

	// Wrap the base transport with DryRunTransport to support context-based dry run
	rclient.SetTransport(&DryRunTransport{
		Transport: baseTransport,
	})

	if opts.UseKeyRing {
		if tok, err := keyring.Get(KeyringName(targetBaseURL, opts.Auth.Tenant), opts.Auth.Username); err == nil {
			if tok != "" {
				slog.Warn("loading token from keyring")
				opts.Auth.Token = tok
			}
		} else {
			slog.Warn("Failed to load token from keyring", "err", err)
		}
	}

	userAgent := defaultUserAgent
	if opts.Agent != "" {
		userAgent = opts.Agent
	}
	rclient.AddRequestMiddleware(MiddlewareAddUserAgent(userAgent, "go-client"))
	rclient.AddRequestMiddleware(MiddlewareAddHost("domain"))
	rclient.AddRequestMiddleware(MiddlewareRemoveEmptyTenantID())
	rclient.SetPathParam("tenantID", opts.Auth.Tenant)
	// rclient.AddRequestMiddleware(MiddlewareAuthorization(opts.Auth))

	c := &Client{
		Client:              rclient,
		BaseURL:             targetBaseURL,
		UserAgent:           userAgent,
		UseTenantInUsername: true,
		Auth:                opts.Auth,

		showSensitive: opts.ShowSensitive,
		UseKeyRing:    opts.UseKeyRing,
	}
	c.common.Client = rclient
	c.common.RealtimeClient = realtime.NewClient(nil, realtime.ClientOptions{
		Host:     targetBaseURL.String(),
		Tenant:   c.Auth.Tenant,
		Username: c.Auth.Username,
		Password: c.Auth.Password,
		Token:    c.Auth.Token,
	})
	c.AuditRecords = auditrecords.NewService(&c.common)
	c.TrustedCertificates = trustedcertificates.NewService(&c.common)
	c.Binaries = binaries.NewService(&c.common)
	c.Identity = identity.NewService(&c.common)
	c.ManagedObjects = managedobjects.NewService(&c.common)

	// Services that use device resolver must be initialized after ManagedObjects
	c.Alarms = alarms.NewService(&c.common, c.ManagedObjects)
	c.Measurements = measurements.NewService(&c.common, c.ManagedObjects)
	c.Operations = operations.NewService(&c.common, c.ManagedObjects)
	c.BulkOperations = bulkoperations.NewService(&c.common)
	c.Events = events.NewService(&c.common, c.ManagedObjects)

	c.Devices = devices.NewService(&c.common)
	c.Applications = applications.NewService(&c.common)
	c.ApplicationVersions = appversions.NewService(&c.common)
	c.Microservices = microservices.NewService(&c.common)
	c.Repository = repository.NewService(&c.common)
	c.UIPlugins = plugins.NewService(&c.common)
	c.UIPluginVersions = versions.NewService(&c.common)
	c.UIApplicationPlugins = applicationplugins.NewService(&c.common)
	c.Notification2 = notification2.NewService(&c.common)
	// c.Context = (*ContextService)(&c.common)
	c.RemoteAccess = remoteaccess.NewService(&c.common)
	c.RetentionRules = retentionrules.NewService(&c.common)
	c.Users = users.NewService(&c.common)
	c.UserGroups = usergroups.NewService(&c.common)
	c.UserRoles = userroles.NewService(&c.common)
	c.Tenants = tenants.NewService(&c.common)
	c.Features = features.NewService(&c.common)
	c.LoginOptions = loginoptions.NewService(&c.common)
	c.LoginTokens = logintokens.NewService(&c.common)

	c.AddMiddleware()

	// Determine the token source.
	// Priority: explicit TokenSource > credential-backed automatic source > static token.
	if opts.Auth.TokenSource != nil {
		c.tokenSource = opts.Auth.TokenSource
	} else if opts.Auth.Username != "" || (opts.Auth.Certificate != "" && opts.Auth.CertificateKey != "") {
		c.tokenSource = c.newInternalTokenSource()
	}

	// Bootstrap: prime the token source / set initial auth.
	// Using a token source: fetch once to populate the cache and set Auth.Token.
	// No token source (static token): set auth directly as before.
	if c.tokenSource != nil {
		if tok, err := c.tokenSource.Token(); err == nil && tok != nil {
			c.SetToken(tok.AccessToken)
			c.SetAuth(c.Auth)
		} else {
			// Either an error (e.g. bad credentials) or nil token (e.g. username
			// present but password empty — no credentials to exchange). In both
			// cases fall back to whatever static auth was supplied (e.g. a raw
			// bearer token).
			if err != nil {
				slog.Debug("Failed to get initial token from token source", "err", err)
			}
			c.SetAuth(opts.Auth)
		}
	} else {
		c.SetAuth(opts.Auth)
	}

	return c
}

// SetBaseURL changes the base url used by the REST client
func (c *Client) AddMiddleware() error {
	// TokenSourceMiddleware injects per-request auth from the active token source.
	// It runs before the global resty auth and overrides it, so callers always get
	// a fresh (non-expired) token without touching global client state.
	c.Client.AddRequestMiddleware(TokenSourceMiddleware(func() authentication.TokenSource {
		return c.tokenSource
	}))
	c.Client.AddRetryConditions(TokenRenewalRetry(c))
	return nil
}
func (c *Client) SetBaseURL(v string) error {
	fmtURL := FormatBaseURL(v)
	targetBaseURL, err := url.Parse(fmtURL)
	if err != nil {
		return err
	}
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.BaseURL = targetBaseURL
	return nil
}

func (c *Client) SetAuth(opt authentication.AuthOptions) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.Auth = opt
	c.updateClientAuth()
}

func (c *Client) updateClientAuth() {
	SetAuth(c.Client, c.Auth)
}

func (c *Client) SetToken(v string) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.Auth.Token = v
	if v != "" {
		if c.UseKeyRing {
			slog.Warn("Trying to save token to keyring")
			if err := keyring.Set(KeyringName(c.BaseURL, c.Auth.Tenant), c.Auth.Username, c.Auth.Token); err != nil {
				slog.Warn("Failed to save token")
			} else {
				slog.Warn("Saved token in keyring")
			}
		}
	}
}

// SetDebug enables or disables custom debug mode with sensitive header redaction.
// This is the safe debug mode that automatically redacts Authorization, Cookie, and other
// sensitive headers before logging.
//
// For debugging authorization issues where you need to see unredacted headers, use
// SetDebugWithAuth instead (but only in local/dev environments).
//
// Example:
//
//	client.SetDebug(true)  // Enable debug with redacted sensitive headers
//	client.SetDebug(false) // Disable debug mode
func (c *Client) SetDebug(enable bool) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	if enable {
		c.showSensitive = false
		c.Client.AddResponseMiddleware(debugLogMiddleware(true))
	} else {
		c.showSensitive = false
	}
}

// SetDebugWithAuth enables debug mode and shows full Authorization headers (unredacted).
// This is useful for local debugging of authentication issues but should NOT be used in production
// as it will expose credentials in logs.
//
// Uses custom middleware to log raw HTTP request/response without header sanitization.
//
// Example:
//
//	client.SetDebugWithAuth(true)  // Shows full auth headers
//	client.SetDebugWithAuth(false) // Disable debug mode
func (c *Client) SetDebugWithAuth(enable bool) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	if enable {
		c.showSensitive = true
		c.Client.AddResponseMiddleware(debugLogMiddleware(false))
		slog.Warn("⚠️  Debug mode enabled with UNREDACTED auth headers - credentials will be visible in logs!")
	} else {
		c.showSensitive = false
	}
}

// debugLogMiddleware returns a response middleware that logs request and response details.
// When sanitize is true, sensitive headers (Authorization, Cookie, etc.) are redacted.
func debugLogMiddleware(sanitize bool) func(*resty.Client, *resty.Response) error {
	return func(client *resty.Client, resp *resty.Response) error {
		req := resp.Request.RawRequest

		fmt.Fprintf(os.Stderr, "\n==============================================================================\n")
		fmt.Fprintf(os.Stderr, "~~~ REQUEST ~~~\n")
		fmt.Fprintf(os.Stderr, "%s  %s  %s\n", req.Method, req.URL.RequestURI(), req.Proto)
		fmt.Fprintf(os.Stderr, "HOST   : %s\n", req.URL.Host)

		// Sanitize headers if requested
		reqHeaders := req.Header
		if sanitize {
			reqHeaders = sanitizeHeaders(req.Header)
		}
		fmt.Fprintf(os.Stderr, "HEADERS:\n%s\n", composeHeaders(reqHeaders))

		// Log request body if available
		if resp.Request.Body != nil {
			contentType := req.Header.Get("Content-Type")
			if isTextContent(contentType) {
				bodyStr := ""
				// Get the actual body bytes
				var bodyBytes []byte
				switch v := resp.Request.Body.(type) {
				case []byte:
					bodyBytes = v
				case string:
					bodyBytes = []byte(v)
				default:
					// For other types (structs, maps, etc.), marshal them
					if b, err := json.Marshal(v); err == nil {
						bodyBytes = b
					} else {
						bodyStr = fmt.Sprintf("%v", v)
					}
				}

				// Pretty-print JSON if we have bytes and content type is JSON
				if len(bodyBytes) > 0 && (strings.Contains(contentType, "application/json") || strings.Contains(contentType, "+json")) {
					var prettyJSON bytes.Buffer
					if err := json.Indent(&prettyJSON, bodyBytes, "", "   "); err == nil {
						bodyStr = prettyJSON.String()
					} else {
						bodyStr = string(bodyBytes)
					}
				} else if len(bodyBytes) > 0 {
					bodyStr = string(bodyBytes)
				}
				fmt.Fprintf(os.Stderr, "BODY   :\n%v\n", bodyStr)
			} else {
				fmt.Fprintf(os.Stderr, "BODY   :\n***** BINARY CONTENT (Content-Type: %s) *****\n", contentType)
			}
		} else {
			fmt.Fprintf(os.Stderr, "BODY   :\n***** NO CONTENT *****\n")
		}

		fmt.Fprintf(os.Stderr, "------------------------------------------------------------------------------\n")
		fmt.Fprintf(os.Stderr, "~~~ RESPONSE ~~~\n")
		fmt.Fprintf(os.Stderr, "STATUS       : %s\n", resp.Status())
		fmt.Fprintf(os.Stderr, "PROTO        : %s\n", resp.Proto())
		fmt.Fprintf(os.Stderr, "RECEIVED AT  : %v\n", resp.ReceivedAt().Format(time.RFC3339Nano))
		fmt.Fprintf(os.Stderr, "DURATION     : %v\n", resp.Duration())

		// Sanitize response headers if requested
		respHeaders := resp.Header()
		if sanitize {
			respHeaders = sanitizeHeaders(resp.Header())
		}
		fmt.Fprintf(os.Stderr, "HEADERS      :\n%s\n", composeHeaders(respHeaders))

		if len(resp.Bytes()) > 0 {
			contentType := resp.Header().Get("Content-Type")
			if isTextContent(contentType) {
				bodyStr := ""
				// Pretty-print JSON if the content type is JSON (including vendor types like +json)
				if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "+json") {
					var prettyJSON bytes.Buffer
					if err := json.Indent(&prettyJSON, resp.Bytes(), "", "   "); err == nil {
						bodyStr = prettyJSON.String()
					} else {
						bodyStr = resp.String()
					}
				} else {
					bodyStr = resp.String()
				}
				fmt.Fprintf(os.Stderr, "BODY         :\n%v\n", bodyStr)
			} else {
				fmt.Fprintf(os.Stderr, "BODY         :\n***** BINARY CONTENT (Content-Type: %s, Size: %d bytes) *****\n", contentType, len(resp.Bytes()))
			}
		} else {
			fmt.Fprintf(os.Stderr, "BODY         :\n***** NO CONTENT *****\n")
		}
		fmt.Fprintf(os.Stderr, "==============================================================================\n")
		return nil
	}
}

// isTextContent checks if a content type is text-based (safe to log)
func isTextContent(contentType string) bool {
	contentType = strings.ToLower(contentType)
	textTypes := []string{
		"text/",
		"application/json",
		"+json", // Vendor-specific JSON types like application/vnd.*+json
		"application/xml",
		"+xml",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
	}
	for _, textType := range textTypes {
		if strings.Contains(contentType, textType) {
			return true
		}
	}
	return false
}

// composeHeaders formats HTTP headers in Resty's debug output style
func composeHeaders(headers http.Header) string {
	if len(headers) == 0 {
		return ""
	}
	var result strings.Builder
	for key, values := range headers {
		for _, value := range values {
			result.WriteString("   ")
			result.WriteString(key)
			result.WriteString(": ")
			result.WriteString(value)
			result.WriteString("\n")
		}
	}
	// Remove trailing newline
	s := result.String()
	if len(s) > 0 {
		s = s[:len(s)-1]
	}
	return s
}

// sanitizeHeaders returns a clone of headers with sensitive values redacted.
// Sensitive headers include Authorization, Cookie, Set-Cookie, and X-Auth-Token.
func sanitizeHeaders(headers http.Header) http.Header {
	if len(headers) == 0 {
		return headers
	}

	sensitiveHeaders := []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
		"X-Auth-Token",
		"X-Api-Key",
		"Proxy-Authorization",
	}

	// Clone headers
	sanitized := make(http.Header, len(headers))
	for key, values := range headers {
		sanitized[key] = make([]string, len(values))
		copy(sanitized[key], values)
	}

	// Redact sensitive headers
	for _, key := range sensitiveHeaders {
		if _, exists := sanitized[key]; exists {
			sanitized[key] = []string{"********************"}
		}
	}

	return sanitized
}

func (c *Client) Login(ctx context.Context) (token string, err error) {
	if len(c.Auth.Certificate) > 0 && len(c.Auth.CertificateKey) > 0 {
		// User certificate
		token, err = c.loginDeviceCertificate(ctx)
	} else if len(c.Auth.Username) > 0 && len(c.Auth.Password) > 0 {
		// Internal SSO
		token, err = c.loginInternalSSO(ctx)
	} else {
		// No login necessary
		return
	}
	if err != nil {
		return
	}
	slog.Debug("Updating client token")
	c.SetToken(token)
	c.SetAuth(c.Auth)
	return
}

func (c *Client) loginInternalSSO(ctx context.Context) (string, error) {
	tok := c.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
		Username:  c.Auth.Username,
		Password:  c.Auth.Password,
		GrantType: logintokens.GrantTypePassword,
	})
	if tok.IsError() {
		return "", tok.Err
	}
	return tok.Data.AccessToken(), nil
}

func (c *Client) loginDeviceCertificate(ctx context.Context) (string, error) {
	tok := c.Devices.CreateAccessToken(ctx)
	if tok.IsError() {
		return "", tok.Err
	}
	return tok.Data.AccessToken(), nil
}

// fetchToken requests a fresh bearer token using the configured credentials.
// It does NOT update Client.Auth or the global resty auth state — the caller is
// responsible for that. ctx should carry WithSkipTokenSource so the login
// request does not re-trigger the TokenSource middleware and cause recursion.
func (c *Client) fetchToken(ctx context.Context) (*authentication.Token, error) {
	var raw string
	var err error
	if len(c.Auth.Certificate) > 0 && len(c.Auth.CertificateKey) > 0 {
		raw, err = c.loginDeviceCertificate(ctx)
	} else if len(c.Auth.Username) > 0 && len(c.Auth.Password) > 0 {
		raw, err = c.loginInternalSSO(ctx)
	} else {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Default expiry — C8Y tokens are typically 1 hour; use 55 min as a buffer.
	expiry := time.Now().Add(55 * time.Minute)
	if claims, parseErr := authentication.ParseToken(raw); parseErr == nil && claims.ExpiresAt != nil {
		expiry = claims.ExpiresAt.Time
	}
	return &authentication.Token{
		AccessToken: raw,
		Expiry:      expiry,
	}, nil
}

// newInternalTokenSource creates a CachedTokenSource backed by this client's
// username/password or device-certificate credentials. The token is refreshed
// automatically when it expires; on a 401, TokenRenewalRetry invalidates the
// cache so the next Token() call fetches a brand-new one.
func (c *Client) newInternalTokenSource() *authentication.CachedTokenSource {
	return authentication.NewCachedTokenSource(
		authentication.TokenSourceFunc(func() (*authentication.Token, error) {
			// Mark the inner login request so TokenSourceMiddleware skips it,
			// preventing an infinite recursion loop.
			ctx := ctxhelpers.WithSkipTokenSource(context.Background())
			tok, err := c.fetchToken(ctx)
			if err != nil || tok == nil {
				return nil, err
			}
			return tok, nil
		}),
	)
}

// HideSensitive checks if sensitive information should be hidden in the logs
func (c *Client) HideSensitive() bool {
	return !c.showSensitive
}

func (c *Client) HideSensitiveInformationIfActive(message string) string {
	// Default to hiding the information
	hideSensitive := c.HideSensitive()
	if v, err := strconv.ParseBool(os.Getenv(EnvVarLoggerHideSensitive)); err == nil {
		hideSensitive = v
	}
	if !hideSensitive {
		return message
	}

	if os.Getenv("USERNAME") != "" {
		message = strings.ReplaceAll(message, os.Getenv("USERNAME"), "******")
	}
	if c.Auth.Tenant != "" {
		message = strings.ReplaceAll(message, c.Auth.Tenant, "{tenant}")
	}
	if c.Auth.Username != "" {
		message = strings.ReplaceAll(message, c.Auth.Username, "{username}")
	}
	if c.Auth.Password != "" {
		message = strings.ReplaceAll(message, c.Auth.Password, "{password}")
	}
	if c.Auth.Token != "" {
		message = strings.ReplaceAll(message, c.Auth.Token, "{token}")
	}

	if c.BaseURL != nil {
		message = strings.ReplaceAll(message, strings.TrimRight(c.BaseURL.Host, "/"), "{host}")
	}
	if c.Domain != "" {
		message = strings.ReplaceAll(message, c.Domain, "{domain}")
	}

	basicAuthMatcher := regexp.MustCompile(`(Basic\s+)[A-Za-z0-9=]+`)
	message = basicAuthMatcher.ReplaceAllString(message, "$1 {base64 tenant/username:password}")

	// bearerAuthMatcher := regexp.MustCompile(`(Bearer\s+)\S+`)
	// message = bearerAuthMatcher.ReplaceAllString(message, "$1 {token}")

	oauthMatcher := regexp.MustCompile(`(authorization=)[^\s]+`)
	message = oauthMatcher.ReplaceAllString(message, "$1{OAuth2Token}")

	xsrfTokenMatcher := regexp.MustCompile(`(?i)((X-)?Xsrf-Token:)\s*[^\s]+`)
	message = xsrfTokenMatcher.ReplaceAllString(message, "$1 {xsrfToken}")

	return message
}

func TokenRenewalRetry(c *Client) func(res *resty.Response, err error) bool {
	return func(res *resty.Response, err error) bool {
		// Network errors (no response) are always retryable
		if err != nil && res == nil {
			return true
		}

		if !res.IsStatusFailure() {
			return false
		}

		statusCode := res.StatusCode()

		// Handle 401 with token renewal
		if statusCode == 401 {
			if res.Request.Attempt > 1 {
				slog.Warn("More than 1 401 detected, giving up", "err", res.ResultError())
				return false
			}

			if res.Request.AuthToken != "" && core.ErrTokenRevoked(res.ResultError()) {
				if c.tokenSource != nil {
					// Force-refresh: if the source supports explicit invalidation (e.g.
					// CachedTokenSource), clear the cache so the next Token() call fetches
					// a brand-new token rather than returning the revoked one.
					if inv, ok := c.tokenSource.(interface{ Invalidate() }); ok {
						inv.Invalidate()
					}
					slog.Warn("Token revoked, refreshing via token source")
					tok, tokErr := c.tokenSource.Token()
					if tokErr != nil || tok == nil {
						return false
					}
					res.Request.SetAuthToken(tok.AccessToken)
					res.Request.Attempt = 0
					return true
				}
				// Fallback for static-token-only clients (no token source configured)
				slog.Warn("Token is not longer valid", "err", res.ResultError())
				loginTok, loginErr := c.Login(res.Request.Context())
				if loginErr != nil {
					return false
				}
				res.Request.SetAuthToken(loginTok)
				res.Request.Attempt = 0
				return true
			}
			return false
		}

		// Only retry on server errors (5xx) and rate limiting (429)
		// Do NOT retry on client errors (4xx) like 404, 400, 403, etc.
		return statusCode == 429 || (statusCode >= 500 && statusCode < 600)
	}

}

func TokenRenewalRetryMiddleware(c *Client) resty.ResponseMiddleware {
	return func(c *resty.Client, r *resty.Response) error {
		slog.Warn("")
		if r.StatusCode() == 401 {

		}
		return nil
	}
}

func KeyringName(host *url.URL, tenant string) string {
	return fmt.Sprintf("go-c8y-%s", strings.Join([]string{host.Hostname(), tenant}, "#"))
}
