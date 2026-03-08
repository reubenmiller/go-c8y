package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base32"
	"encoding/binary"
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
	"sync/atomic"
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
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/devicestatistics"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/applicationplugins"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/plugins"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/ui/plugins/versions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles"
	inventoryroles "github.com/reubenmiller/go-c8y/pkg/c8y/api/userroles/inventory"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/currentuser/totp"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/users/devicepermissions"
	"github.com/reubenmiller/go-c8y/pkg/certutil"
	"github.com/zalando/go-keyring"
	"resty.dev/v3"
)

var ErrNotFound = errors.New("item: not found")

// ErrNoLoginMethodAvailable is returned by LoginWithOptions when a Preference
// list is given but no method in the list could be used (all were either missing
// the required credentials or skipped because the tenant has no external OAuth2
// provider configured).
var ErrNoLoginMethodAvailable = errors.New("login: no method available (all preferences exhausted)")

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
	clientMu   sync.Mutex    // clientMu protects the client during calls that modify the CheckRedirect func.
	HTTPClient *resty.Client // HTTP client used to communicate with the API.

	RealtimeClient *realtime.Client

	// Show sensitive information
	showSensitive bool

	// debugEnabled controls whether the debug response middleware produces output.
	debugEnabled bool

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
	InventoryRoles       *inventoryroles.Service
	DevicePermissions    *devicepermissions.Service
	DeviceStatistics     *devicestatistics.Service
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

	// Show sensitive information in the logs (e.g. credentials in Authorization headers).
	// When true, sensitive headers are NOT redacted in debug output.
	// Has no effect unless Debug is also true.
	ShowSensitive bool

	// Debug enables debug logging of all HTTP requests and responses made by this
	// client, including any calls made during NewClient() itself (e.g. the initial
	// token fetch via certificate auth). Sensitive headers are redacted unless
	// ShowSensitive is also set.
	Debug bool

	// Enable Keyring for saving and retrieving a token
	UseKeyRing bool

	// Not recommended
	InsecureSkipVerify bool

	Agent string

	Transport http.RoundTripper

	// Timeout sets a global HTTP-level timeout for all requests made by this client.
	// A value of zero means no timeout. Per-request timeouts can still be applied
	// by passing a context with a deadline: ctx, cancel := context.WithTimeout(ctx, d)
	Timeout time.Duration

	// MTLSPort is the port used for the mutual-TLS device-certificate token endpoint.
	// Defaults to "8443" when empty. This is the port on which Cumulocity listens for
	// mTLS connections (e.g. POST /devicecontrol/deviceAccessToken).
	MTLSPort string
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

// buildCertChainHeader returns the value for the X-SSL-CERT-CHAIN request header
// derived from the intermediate certificates in the supplied auth credentials.
// Returns an empty string when no certificate is configured or when there are no
// intermediate certificates in the chain (i.e. the leaf cert was issued directly
// by a root that is already trusted by the platform).
func buildCertChainHeader(auth authentication.AuthOptions) string {
	if auth.Certificate == "" || auth.CertificateKey == "" {
		return ""
	}
	var cert tls.Certificate
	var err error
	if _, statErr := os.Stat(auth.CertificateKey); statErr == nil {
		cert, err = tls.LoadX509KeyPair(auth.Certificate, auth.CertificateKey)
	} else {
		cert, err = tls.X509KeyPair([]byte(auth.Certificate), []byte(auth.CertificateKey))
	}
	if err != nil {
		return ""
	}
	headerValue, err := certutil.CertificateChain([]tls.Certificate{cert}).Header()
	if err != nil || len(headerValue) == 0 {
		return ""
	}
	return string(headerValue)
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

	if opts.Timeout > 0 {
		rclient.SetTimeout(opts.Timeout)
	}

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
		tlsCfg := &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
		}
		// Attach client certificates directly to the transport for mTLS (e.g.
		// device certificate auth on port 8443). SetCertificateChainHeaderIfRequired
		// configures resty's own internal TLS state, but the SetTransport call below
		// replaces the transport entirely and loses those settings. We must therefore
		// also set the certificates on the transport we are about to install.
		if opts.Auth.Certificate != "" && opts.Auth.CertificateKey != "" {
			var cert tls.Certificate
			var certErr error
			if _, statErr := os.Stat(opts.Auth.CertificateKey); statErr == nil {
				cert, certErr = tls.LoadX509KeyPair(opts.Auth.Certificate, opts.Auth.CertificateKey)
			} else {
				cert, certErr = tls.X509KeyPair([]byte(opts.Auth.Certificate), []byte(opts.Auth.CertificateKey))
			}
			if certErr == nil {
				tlsCfg.Certificates = []tls.Certificate{cert}
			} else {
				slog.Debug("Failed to load client certificate onto transport", "err", certErr)
			}
		}
		defaultTransport.TLSClientConfig = tlsCfg
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
	rclient.SetPathParam(core.PathParamTenantID, opts.Auth.Tenant)
	// rclient.AddRequestMiddleware(MiddlewareAuthorization(opts.Auth))

	c := &Client{
		HTTPClient:          rclient,
		BaseURL:             targetBaseURL,
		UserAgent:           userAgent,
		UseTenantInUsername: true,
		Auth:                opts.Auth,

		debugEnabled:  opts.Debug,
		showSensitive: opts.ShowSensitive,
		UseKeyRing:    opts.UseKeyRing,
	}
	c.common.Client = rclient
	c.common.MTLSPort = opts.MTLSPort
	c.common.CertChainHeader = buildCertChainHeader(opts.Auth)
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
	c.InventoryRoles = inventoryroles.NewService(&c.common)
	c.DevicePermissions = devicepermissions.NewService(&c.common)
	c.DeviceStatistics = devicestatistics.NewService(&c.common)
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
	c.HTTPClient.AddRequestMiddleware(TokenSourceMiddleware(func() authentication.TokenSource {
		return c.tokenSource
	}))
	c.HTTPClient.AddRetryConditions(TokenRenewalRetry(c))

	// Register a single debug middleware that reads the runtime debugEnabled flag.
	// This avoids the problem of middleware accumulating each time SetDebug is called.
	c.HTTPClient.AddResponseMiddleware(func(rc *resty.Client, resp *resty.Response) error {
		if !c.debugEnabled {
			return nil
		}
		return debugLogMiddleware(!c.showSensitive)(rc, resp)
	})
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
	SetAuth(c.HTTPClient, c.Auth)
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

	c.debugEnabled = enable
	c.showSensitive = false
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

	c.debugEnabled = enable
	c.showSensitive = enable
	if enable {
		slog.Warn("⚠️  Debug mode enabled with UNREDACTED auth headers - credentials will be visible in logs!")
	}
}

// clientTLSCerts returns the client certificates configured on the resty client's
// transport, if any. These are the certificates presented during mTLS handshakes.
// Duplicates (same leaf certificate bytes) are removed.
func clientTLSCerts(client *resty.Client) []tls.Certificate {
	var tlsCfg *tls.Config
	switch t := client.Transport().(type) {
	case *DryRunTransport:
		tlsCfg = t.TLSClientConfig()
	case *http.Transport:
		tlsCfg = t.TLSClientConfig
	case interface{ TLSClientConfig() *tls.Config }:
		tlsCfg = t.TLSClientConfig()
	}
	if tlsCfg == nil {
		return nil
	}
	// Deduplicate by leaf certificate bytes so the same cert isn't printed twice
	// (can happen when the cert is registered in multiple places in the TLS config).
	seen := make(map[string]struct{}, len(tlsCfg.Certificates))
	unique := make([]tls.Certificate, 0, len(tlsCfg.Certificates))
	for _, cert := range tlsCfg.Certificates {
		if len(cert.Certificate) == 0 {
			continue
		}
		key := string(cert.Certificate[0])
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		unique = append(unique, cert)
	}
	return unique
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

		// Show client certificate info for mTLS requests.
		if certs := clientTLSCerts(client); len(certs) > 0 {
			fmt.Fprintf(os.Stderr, "CLIENT CERTS:\n")
			for _, cert := range certs {
				if len(cert.Certificate) == 0 {
					continue
				}
				if x509Cert, err := x509.ParseCertificate(cert.Certificate[0]); err == nil {
					fmt.Fprintf(os.Stderr, "   Subject    : %s\n", x509Cert.Subject.String())
					fmt.Fprintf(os.Stderr, "   Issuer     : %s\n", x509Cert.Issuer.String())
					fmt.Fprintf(os.Stderr, "   Valid From : %s\n", x509Cert.NotBefore.Format(time.RFC3339))
					fmt.Fprintf(os.Stderr, "   Expires    : %s\n", x509Cert.NotAfter.Format(time.RFC3339))
				}
			}
		}

		// Sanitize headers if requested
		reqHeaders := req.Header
		if sanitize {
			reqHeaders = sanitizeHeaders(req.Header)
		}
		fmt.Fprintf(os.Stderr, "HEADERS:\n%s\n", composeHeaders(reqHeaders))

		// Log request body if available
		contentType := req.Header.Get("Content-Type")
		if resp.Request.Body != nil {
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
		} else if len(resp.Request.FormData) > 0 {
			// resty stores form-encoded fields in FormData, not Body
			fmt.Fprintf(os.Stderr, "BODY   :\n%s\n", resp.Request.FormData.Encode())
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

		respBytes := resp.Bytes()
		// If resty auto-unmarshaled the body into Result/ResultError, bodyBytes is
		// consumed. Fall back to marshaling the parsed struct so the debug output
		// is still useful.
		if len(respBytes) == 0 && resp.IsRead {
			var data any
			if resp.IsStatusFailure() {
				data = resp.ResultError()
			} else {
				data = resp.Result()
			}
			if data != nil {
				if b, err := json.MarshalIndent(data, "", "   "); err == nil {
					respBytes = b
				}
			}
		}
		if len(respBytes) > 0 {
			contentType := resp.Header().Get("Content-Type")
			if isTextContent(contentType) || json.Valid(respBytes) {
				bodyStr := ""
				// Pretty-print JSON if the content type is JSON (including vendor types like +json)
				if strings.Contains(contentType, "application/json") || strings.Contains(contentType, "+json") || json.Valid(respBytes) {
					var prettyJSON bytes.Buffer
					if err := json.Indent(&prettyJSON, respBytes, "", "   "); err == nil {
						bodyStr = prettyJSON.String()
					} else {
						bodyStr = string(respBytes)
					}
				} else {
					bodyStr = string(respBytes)
				}
				fmt.Fprintf(os.Stderr, "BODY         :\n%v\n", bodyStr)
			} else {
				fmt.Fprintf(os.Stderr, "BODY         :\n***** BINARY CONTENT (Content-Type: %s, Size: %d bytes) *****\n", contentType, len(respBytes))
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

// SMSChallenge is passed to the LoginOptions.SMSCode callback when Cumulocity
// signals that an SMS PIN has been sent to the user (a 401 whose error message
// matches "pin.*generated").
type SMSChallenge struct {
	// Message is the lower-cased error message from the 401 response.
	Message string
}

// PasswordChangeChallenge is passed to the LoginOptions.PasswordChange callback
// when Cumulocity forces the user to set a new password at login time (a 401
// with a non-empty "passwordresettoken" response header).
type PasswordChangeChallenge struct {
	// Token is the one-time reset token from the "passwordresettoken" header.
	// It must be forwarded as-is when calling the password-reset API.
	Token string
	// Email is the pre-populated value from c.Auth.Username. It is provided as
	// a hint only — the username is not always the user's email address, so the
	// callback should confirm or prompt for the actual email before using it.
	Email string
	// Message is the lower-cased error message from the 401 response.
	Message string
}

// LoginOptions configures optional behaviour for LoginWithOptions.
type LoginOptions struct {
	// Method enforces a specific login flow. LoginWithOptions fails immediately
	// if the method requires credentials that are not set, or if the tenant has
	// no external OAuth2 provider (for SSO-based methods).
	// Mutually exclusive with Preference — set one or neither.
	Method authentication.LoginMethod

	// Preference is an ordered list of login flows to try. Each method is
	// evaluated for local availability (credentials present, etc.); the first
	// that is satisfied is attempted. SSO-based methods (device flow, browser
	// flow) are silently skipped when the tenant has no external provider.
	// When both Method and Preference are unset, the default behaviour is
	// preserved: OAUTH2_INTERNAL is attempted with full TFA callback support.
	Preference []authentication.LoginMethod

	// BrowserFlow configures the SSO Authorization Code flow. Used when
	// LoginMethodOAuth2BrowserFlow is selected via Method or Preference.
	// When nil, sane defaults are applied (listen on 127.0.0.1:5001, open
	// the system browser).
	BrowserFlow *BrowserFlowOptions

	// DeviceFlow configures the SSO Device Authorization flow. Used when
	// LoginMethodOAuth2DeviceFlow is selected via Method or Preference.
	// When nil, OAuth2 endpoints are auto-discovered and the code is printed
	// to stderr.
	DeviceFlow *DeviceFlowOptions

	// TOTPSecret is a base32-encoded TOTP secret used to generate codes
	// automatically via RFC 6238 (HMAC-SHA1, 30-second window). When set,
	// TOTPCode is ignored for code generation and the code is computed from
	// this secret instead.
	//
	// WARNING: this is intended for machine/automation scenarios only.
	// Storing a TOTP secret alongside credentials eliminates the second-factor
	// security benefit and is NOT recommended for interactive users.
	TOTPSecret string

	// TOTPCode is called whenever the login flow needs a TOTP code: either
	// during a first-time enrollment verification or a normal login challenge.
	// Ignored when TOTPSecret is set. If nil and TOTPSecret is not set, the
	// flow will return an error when a code is required.
	TOTPCode totp.TOTPCodeFunc

	// QRCode is called with the otpauth:// URL and the raw TOTP secret when a
	// new TOTP secret is generated, so the caller can render a QR code in the
	// terminal or UI, or display the secret for manual entry.
	// If nil, the URL is silently skipped.
	QRCode func(otpauthURL string, secret string)

	// SMSCode is called when the server issues an SMS PIN challenge (a 401
	// whose message matches "pin.*generated"). The function should prompt the
	// user for the PIN received via SMS and return it.
	// If nil, an SMS challenge surfaces as an error.
	SMSCode func(ctx context.Context, challenge SMSChallenge) (string, error)

	// PasswordChange is called when the server forces a password change at
	// login time (a 401 with a non-empty "passwordresettoken" response header).
	// The function should prompt the user for their email and new password and
	// return both. challenge.Email is pre-populated with the login username as a
	// hint, but it may not be the user's actual email address.
	// If nil, a forced-password-change challenge surfaces as an error.
	PasswordChange func(ctx context.Context, challenge PasswordChangeChallenge) (email string, newPassword string, err error)

	// CredentialPrompt is called when a method that requires credentials is
	// about to be attempted but those credentials are missing from the client's
	// auth state. The callback receives the method being attempted and a
	// pointer to the auth options so it can fill in whatever is missing:
	//
	//   BASIC / OAUTH2_INTERNAL  →  auth.Username, auth.Password
	//   CERTIFICATE              →  auth.Certificate, auth.CertificateKey
	//
	// The updated values are applied to the client before the login attempt
	// proceeds. If the callback returns a non-nil error, the login fails with
	// that error. If nil and credentials are still missing, the method fails
	// with its usual credential-absent error.
	//
	// In Preference mode, returning a non-nil error from CredentialPrompt
	// surfaces immediately rather than moving to the next preference.
	CredentialPrompt func(ctx context.Context, method authentication.LoginMethod, auth *authentication.AuthOptions) error

	// OnSuccess is called after a successful login with the method that was
	// actually used. Useful when Preference is set and the caller wants to
	// know which method in the list won. When neither Method nor Preference
	// is set the reported method is LoginMethodOAuth2Internal.
	// If nil, the callback is skipped.
	OnSuccess func(method authentication.LoginMethod)
}

// LoginWithOptions performs a Cumulocity login using the flow specified by
// opts.Method or opts.Preference. When neither is set the behaviour is
// identical to the pre-preference behaviour: OAUTH2_INTERNAL is attempted
// with full TFA and forced-password-change callback support.
//
// Method vs. Preference:
//
//   - Method (strict): exactly one flow is attempted; login fails immediately
//     if the required credentials are absent or the tenant has no external
//     OAuth2 provider.
//   - Preference (soft): methods are tried in order; a method is skipped
//     when its local prerequisites are not met (username/password absent,
//     certificate files missing), or when the tenant has no external OAuth2
//     provider for SSO flows. Returns ErrNoLoginMethodAvailable if every
//     method in the list is skipped.
//
// On success the client's auth state is updated so that subsequent API calls
// use the obtained token or credential type.
func (c *Client) LoginWithOptions(ctx context.Context, opts LoginOptions) (token string, err error) {
	// Legacy / default path: no explicit method requested.
	if opts.Method == "" && len(opts.Preference) == 0 {
		token, err = c.loginInternalSSOWithTFA(ctx, opts)
		if err == nil && opts.OnSuccess != nil {
			opts.OnSuccess(authentication.LoginMethodOAuth2Internal)
		}
		return
	}

	strict := opts.Method != ""
	methods := opts.Preference
	if strict {
		methods = []authentication.LoginMethod{opts.Method}
	}

	for _, m := range methods {
		// In preference mode, skip methods whose local prerequisites are absent.
		// Exception: if a CredentialPrompt is registered, don't skip credential-
		// based methods just because credentials are currently absent —
		// loginWithMethod will call the prompt to fill them in.
		if !strict && !c.isLoginMethodAvailable(m) {
			isCredentialMethod := m == authentication.LoginMethodBasic ||
				m == authentication.LoginMethodOAuth2Internal ||
				m == authentication.LoginMethodCertificate
			if opts.CredentialPrompt == nil || !isCredentialMethod {
				continue
			}
		}
		token, err = c.loginWithMethod(ctx, m, opts)
		if err == nil {
			if opts.OnSuccess != nil {
				opts.OnSuccess(m)
			}
			return
		}
		// In preference mode, treat "no external provider" as "method
		// unavailable" and continue to the next preference.
		if !strict && errors.Is(err, core.ErrNoAuth2Provider) {
			err = nil
			continue
		}
		// Any other error — surface it (both strict and preference).
		return
	}

	// All preferences were exhausted without a successful login.
	if err == nil {
		err = ErrNoLoginMethodAvailable
	}
	return
}

// isLoginMethodAvailable reports whether the local configuration is sufficient
// to attempt method m, without making any network calls.
func (c *Client) isLoginMethodAvailable(m authentication.LoginMethod) bool {
	switch m {
	case authentication.LoginMethodBasic, authentication.LoginMethodOAuth2Internal:
		return c.Auth.Username != "" && c.Auth.Password != ""
	case authentication.LoginMethodCertificate:
		return c.Auth.Certificate != "" && c.Auth.CertificateKey != ""
	case authentication.LoginMethodOAuth2DeviceFlow, authentication.LoginMethodOAuth2BrowserFlow:
		// SSO availability requires a network call; assume available and let
		// AuthorizeWith* return ErrNoAuth2Provider if the tenant lacks a provider.
		return true
	default:
		return false
	}
}

// loginWithMethod runs the single login flow identified by m.
func (c *Client) loginWithMethod(ctx context.Context, m authentication.LoginMethod, opts LoginOptions) (string, error) {
	// If the method needs credentials and some are missing, give the caller a
	// chance to fill them in interactively before we check.
	if opts.CredentialPrompt != nil {
		needsPrompt := false
		switch m {
		case authentication.LoginMethodBasic, authentication.LoginMethodOAuth2Internal:
			needsPrompt = c.Auth.Username == "" || c.Auth.Password == ""
		case authentication.LoginMethodCertificate:
			needsPrompt = c.Auth.Certificate == "" || c.Auth.CertificateKey == ""
		}
		if needsPrompt {
			auth := c.Auth
			if err := opts.CredentialPrompt(ctx, m, &auth); err != nil {
				return "", err
			}
			c.Auth = auth
			c.SetAuth(auth)
		}
	}

	switch m {
	case authentication.LoginMethodBasic:
		if c.Auth.Username == "" || c.Auth.Password == "" {
			return "", fmt.Errorf("basic auth: username and password are required")
		}
		// Basic auth requires no token exchange; just pin the auth type so
		// the middleware always sends an Authorization: Basic header.
		c.Auth.AuthType = []authentication.AuthType{authentication.AuthTypeBasic}
		c.SetAuth(c.Auth)
		return "", nil

	case authentication.LoginMethodOAuth2Internal:
		return c.loginInternalSSOWithTFA(ctx, opts)

	case authentication.LoginMethodCertificate:
		tok, err := c.loginDeviceCertificate(ctx)
		if err != nil {
			return "", err
		}
		c.SetToken(tok)
		c.SetAuth(c.Auth)
		return tok, nil

	case authentication.LoginMethodOAuth2DeviceFlow:
		d := DeviceFlowOptions{}
		if opts.DeviceFlow != nil {
			d = *opts.DeviceFlow
		}
		accessToken, err := c.AuthorizeWithDeviceFlow(ctx, "", d.AuthEndpoints, d.DisplayFunc)
		if err != nil {
			return "", err
		}
		return accessToken.Token, nil

	case authentication.LoginMethodOAuth2BrowserFlow:
		bOpts := BrowserFlowOptions{}
		if opts.BrowserFlow != nil {
			bOpts = *opts.BrowserFlow
		}
		accessToken, err := c.AuthorizeWithBrowserFlow(ctx, "", bOpts)
		if err != nil {
			return "", err
		}
		return accessToken.Token, nil

	default:
		return "", fmt.Errorf("login: unknown method %q", m)
	}
}

// loginInternalSSOWithTFA performs an OAUTH2_INTERNAL login with full
// TOTP flow, SMS TFA challenges, and forced password-change handling:
//
//  1. Attempts a normal OAI-Secure token request.
//  2. If the server requires TOTP setup:
//     a. Calls GenerateSecret to obtain the TOTP secret.
//     b. Invokes opts.QRCode (if set) with the otpauth:// enrolment URL.
//     c. Calls opts.TOTPCode to get a verification code from the user.
//     d. Verifies the code, then activates TOTP.
//     e. Retries the login with the same code as tfa_code.
//  3. If the server issues a TOTP challenge (already enrolled):
//     a. Calls opts.TOTPCode to get the current TOTP code.
//     b. Retries the login with tfa_code set.
//  4. If the server returns a 401 with a "passwordresettoken" header:
//     a. Calls opts.PasswordChange to get a new password from the user.
//     b. Calls the password-reset API, then re-logs in with the new password.
//  5. If the server returns a 401 with an SMS pin message:
//     a. Calls opts.SMSCode to get the PIN from the user.
//     b. Retries the login with tfa_code set to the SMS PIN.
//  6. On success, stores the token and updates the client auth state.
func (c *Client) loginInternalSSOWithTFA(ctx context.Context, opts LoginOptions) (token string, err error) {
	token, err = c.loginInternalSSO(ctx)
	if err == nil {
		c.seedTokenSource(token)
		return
	}

	if opts.TOTPSecret == "" && opts.TOTPCode == nil && opts.SMSCode == nil && opts.PasswordChange == nil {
		// No handler — surface the error as-is.
		return
	}

	var apiErr *core.Error
	if !errors.As(err, &apiErr) {
		return
	}

	errorMessage := strings.ToLower(apiErr.Error())
	var tfaCode string

	switch {
	case apiErr.StatusCode() == 401 && strings.Contains(errorMessage, "totp setup required"):
		// First-time enrollment: generate secret → show QR → verify → activate.
		secretResult := c.Users.CurrentUser.TOTP.GenerateSecret(ctx)
		if secretResult.Err != nil {
			return "", fmt.Errorf("totp: generate secret: %w", secretResult.Err)
		}

		if opts.QRCode != nil {
			rawSecret := secretResult.Data.RawSecret()
			otpauthURL := fmt.Sprintf(
				"otpauth://totp/%s?secret=%s&issuer=%s",
				c.Auth.Username,
				rawSecret,
				c.BaseURL.Host,
			)
			opts.QRCode(otpauthURL, rawSecret)
		}

		tfaCode, err = c.totpCodeFrom(ctx, opts, totp.TOTPChallenge{
			IsSetup: true,
			Message: errorMessage,
		})
		if err != nil {
			return "", fmt.Errorf("totp: code input: %w", err)
		}

		if r := c.Users.CurrentUser.TOTP.VerifyCode(ctx, tfaCode); r.Err != nil {
			return "", fmt.Errorf("totp: verify code: %w", r.Err)
		}
		if r := c.Users.CurrentUser.TOTP.SetActivity(ctx, true); r.Err != nil {
			return "", fmt.Errorf("totp: activate: %w", r.Err)
		}

	case apiErr.StatusCode() == 401 && strings.Contains(errorMessage, "totp"):
		// Ongoing login challenge: user already enrolled, just needs to supply code.
		tfaCode, err = c.totpCodeFrom(ctx, opts, totp.TOTPChallenge{
			IsSetup: false,
			Message: errorMessage,
		})
		if err != nil {
			return "", fmt.Errorf("totp: code input: %w", err)
		}

	case apiErr.StatusCode() == 401 &&
		apiErr.Response != nil &&
		apiErr.Response.Header.Get("passwordresettoken") != "":
		// The server is forcing a password change before allowing login.
		if opts.PasswordChange == nil {
			return
		}
		resetToken := apiErr.Response.Header.Get("passwordresettoken")
		var email, newPassword string
		email, newPassword, err = opts.PasswordChange(ctx, PasswordChangeChallenge{
			Token:   resetToken,
			Email:   c.Auth.Username,
			Message: errorMessage,
		})
		if err != nil {
			return "", fmt.Errorf("password change: %w", err)
		}
		if r := c.Users.ResetPassword(ctx, users.ResetPasswordOptions{
			Tenant:           c.Auth.Tenant,
			Token:            resetToken,
			Email:            email,
			NewPassword:      newPassword,
			PasswordStrength: users.CalculatePasswordStrength(newPassword),
		}); r.Err != nil {
			return "", fmt.Errorf("password change: reset: %w", r.Err)
		}
		c.Auth.Password = newPassword
		// Re-run the full login flow so that any additional challenges
		// (TOTP, SMS TFA) that follow the password change are handled too.
		token, err = c.LoginWithOptions(ctx, opts)
		if err != nil {
			return "", fmt.Errorf("password change: re-login: %w", err)
		}
		return

	case apiErr.StatusCode() == 401 &&
		strings.Contains(errorMessage, "pin") &&
		strings.Contains(errorMessage, "generated"):
		// The server has sent an SMS PIN to the user's registered phone.
		if opts.SMSCode == nil {
			return
		}
		var smsCode string
		smsCode, err = opts.SMSCode(ctx, SMSChallenge{Message: errorMessage})
		if err != nil {
			return "", fmt.Errorf("sms: code input: %w", err)
		}
		tfaCode = smsCode

	default:
		// Unrelated 401 — surface as-is.
		return
	}

	// Retry login with the TOTP code supplied.
	tok := c.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
		Tenant:    c.Auth.Tenant,
		Username:  c.Auth.Username,
		Password:  c.Auth.Password,
		TFACode:   tfaCode,
		GrantType: logintokens.GrantTypePassword,
	})
	if tok.IsError() {
		return "", tok.Err
	}
	token = tok.Data.AccessToken()
	c.seedTokenSource(token)
	return
}

// totpCodeFrom resolves a TOTP code for the given challenge using opts.
// If opts.TOTPSecret is set the code is computed automatically via RFC 6238;
// otherwise opts.TOTPCode is invoked interactively.
func (c *Client) totpCodeFrom(ctx context.Context, opts LoginOptions, challenge totp.TOTPChallenge) (string, error) {
	if secret := strings.TrimSpace(opts.TOTPSecret); secret != "" {
		return computeTOTPCode(secret)
	}
	if opts.TOTPCode == nil {
		return "", fmt.Errorf("totp: no code source configured (set TOTPSecret or TOTPCode)")
	}
	return opts.TOTPCode(ctx, challenge)
}

// computeTOTPCode generates a 6-digit TOTP code from a base32-encoded secret
// using RFC 6238 (HMAC-SHA1, 30-second window). It accepts secrets with or
// without standard padding and is case-insensitive.
func computeTOTPCode(secret string) (string, error) {
	secret = strings.ToUpper(strings.ReplaceAll(secret, " ", ""))
	if pad := len(secret) % 8; pad != 0 {
		secret += strings.Repeat("=", 8-pad)
	}
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", fmt.Errorf("totp: invalid base32 secret: %w", err)
	}
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(time.Now().Unix())/30)
	mac := hmac.New(sha1.New, key)
	mac.Write(counter[:])
	h := mac.Sum(nil)
	offset := h[len(h)-1] & 0x0f
	code := (uint32(h[offset])&0x7f)<<24 |
		uint32(h[offset+1])<<16 |
		uint32(h[offset+2])<<8 |
		uint32(h[offset+3])
	return fmt.Sprintf("%06d", code%1_000_000), nil
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

// seedTokenSource pre-populates the CachedTokenSource cache with a token that
// was just acquired externally (e.g. via a TOTP-gated login), so the next API
// call can use it directly instead of triggering an unnecessary token fetch.
// It also stores the token in Auth and updates the resty client auth state.
func (c *Client) seedTokenSource(raw string) {
	if cached, ok := c.tokenSource.(*authentication.CachedTokenSource); ok {
		expiry := time.Now().Add(55 * time.Minute)
		if claims, parseErr := authentication.ParseToken(raw); parseErr == nil && claims.ExpiresAt != nil {
			expiry = claims.ExpiresAt.Time
		}
		cached.Seed(&authentication.Token{AccessToken: raw, Expiry: expiry})
	}
	c.SetToken(raw)
	c.SetAuth(c.Auth)
}

// newInternalTokenSource creates a CachedTokenSource backed by this client's
// username/password or device-certificate credentials. The token is refreshed
// automatically when it expires; on a 401, TokenRenewalRetry invalidates the
// cache so the next Token() call fetches a brand-new one.
//
// For username/password: if the token exchange fails (e.g. the tenant has
// OAUTH2_INTERNAL disabled), the source marks itself unavailable and returns
// (nil, nil) so that the resty client-level basic-auth credentials take over
// instead of every request failing with a "token source" error.
//
// For certificate auth: there is no basic-auth fallback, so errors are
// propagated. CachedTokenSource never caches a failed fetch, meaning the
// source retries on every subsequent request until the mTLS endpoint responds.
func (c *Client) newInternalTokenSource() *authentication.CachedTokenSource {
	certAuth := c.Auth.Certificate != "" && c.Auth.CertificateKey != ""
	var oauth2Unavailable atomic.Bool
	return authentication.NewCachedTokenSource(
		authentication.TokenSourceFunc(func() (*authentication.Token, error) {
			if oauth2Unavailable.Load() {
				return nil, nil
			}
			// Mark the inner login request so TokenSourceMiddleware skips it,
			// preventing an infinite recursion loop.
			ctx := ctxhelpers.WithSkipTokenSource(context.Background())
			tok, err := c.fetchToken(ctx)
			if err != nil {
				if certAuth {
					// For certificate-only auth there is no basic-auth fallback.
					// Propagate the error so CachedTokenSource does not cache the
					// failure and retries on the next request.
					return nil, err
				}
				slog.Debug("OAuth2 internal token exchange failed, falling back to basic auth", "err", err)
				oauth2Unavailable.Store(true)
				return nil, nil // Let resty client-level basic auth handle it
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
