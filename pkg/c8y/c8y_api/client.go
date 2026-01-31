package c8y_api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/applications"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/auditrecords"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/devices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/events"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/features"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/identity"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/loginoptions"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/microservices"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/operations"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/repository"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/retentionrules"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/tenants/logintokens"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/trustedcertificates"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/userroles"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/users"
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

	UseKeyRing bool

	// Cumulocity Version
	Version string

	UseTenantInUsername bool

	common core.Service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the Cumulocity API.
	// Context              *ContextService
	Alarms       *alarms.Service
	AuditRecords *auditrecords.Service
	// DeviceCredentials    *DeviceCredentialsService
	Measurements *measurements.Service
	Binaries     *binaries.Service

	LoginOptions *loginoptions.Service

	LoginTokens *logintokens.Service

	Devices        *devices.Service
	ManagedObjects *managedobjects.Service
	Operations     *operations.Service
	Tenants        *tenants.Service
	Events         *events.Service
	Applications   *applications.Service
	Microservices  *microservices.Service
	Repository     *repository.Service
	// UIExtension          *UIExtensionService
	// ApplicationVersions  *ApplicationVersionsService
	Identity            *identity.Service
	TrustedCertificates *trustedcertificates.Service
	// Notification2        *Notification2Service
	// RemoteAccess         *RemoteAccessService
	RetentionRules *retentionrules.Service
	// Software             *InventorySoftwareService
	// Firmware             *InventoryFirmwareService
	Users      *users.Service
	UserGroups *usergroups.Service
	UserRoles  *userroles.Service
	// DeviceEnrollment     *DeviceEnrollmentService
	Features *features.Service
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
}

func NewClientFromEnvironment(opt ClientOptions) *Client {
	opt.BaseURL = authentication.HostFromEnvironment()
	opt.Auth = authentication.FromEnvironment()
	return NewClient(opt)
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
		SetError(core.APIError{}).
		SetCircuitBreaker(circuitBreaker)

	targetBaseURL, _ := url.Parse(FormatBaseURL(opts.BaseURL))
	if targetBaseURL != nil {
		rclient.SetBaseURL(targetBaseURL.String())
	}

	rclient.TLSClientConfig().InsecureSkipVerify = opts.InsecureSkipVerify

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
	c.Alarms = (*alarms.Service)(&c.common)
	c.AuditRecords = (*auditrecords.Service)(&c.common)
	c.TrustedCertificates = trustedcertificates.NewService(&c.common)
	// c.DeviceCredentials = (*DeviceCredentialsService)(&c.common)
	c.Measurements = (*measurements.Service)(&c.common)
	c.Binaries = (*binaries.Service)(&c.common)
	c.Identity = identity.NewService(&c.common)
	c.ManagedObjects = managedobjects.NewService(&c.common)
	c.Devices = devices.NewService(&c.common)
	c.LoginOptions = loginoptions.NewService(&c.common)
	c.LoginTokens = logintokens.NewService(&c.common)
	c.Operations = (*operations.Service)(&c.common)
	c.Tenants = tenants.NewService(&c.common)
	c.Events = events.NewService(&c.common)
	// c.DeviceEnrollment = (*DeviceEnrollmentService)(&c.common)
	c.Applications = applications.NewService(&c.common)
	c.Microservices = microservices.NewService(&c.common)
	c.Repository = repository.NewService(&c.common)
	// c.ApplicationVersions = (*ApplicationVersionsService)(&c.common)
	// c.UIExtension = (*UIExtensionService)(&c.common)
	// c.Notification2 = (*Notification2Service)(&c.common)
	// c.Context = (*ContextService)(&c.common)
	// c.RemoteAccess = (*RemoteAccessService)(&c.common)
	c.RetentionRules = retentionrules.NewService(&c.common)
	// c.Software = (*InventorySoftwareService)(&c.common)
	// c.Firmware = (*InventoryFirmwareService)(&c.common)
	c.Users = users.NewService(&c.common)
	c.UserGroups = usergroups.NewService(&c.common)
	c.UserRoles = userroles.NewService(&c.common)
	c.Features = features.NewService(&c.common)
	c.AddMiddleware()
	c.SetAuth(opts.Auth)
	if _, err := c.Login(context.Background()); err != nil {
		slog.Debug("Failed to get a token", "err", err)
	}
	return c
}

// SetBaseURL changes the base url used by the REST client
func (c *Client) AddMiddleware() error {

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
	tok, err := c.LoginTokens.Create(ctx, logintokens.CreateTokenOptions{
		Username:  c.Auth.Username,
		Password:  c.Auth.Password,
		GrantType: logintokens.GrantTypePassword,
	})
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func (c *Client) loginDeviceCertificate(ctx context.Context) (string, error) {
	tok, err := c.Devices.CreateAccessToken(ctx)
	if err != nil {
		return "", err
	}
	return tok.AccessToken, nil
}

func TokenRenewalRetry(c *Client) func(res *resty.Response, err error) bool {
	return func(res *resty.Response, err error) bool {
		if !res.IsError() {
			return false
		}
		if res.StatusCode() == 401 {
			if res.Request.Attempt > 1 {
				slog.Warn("More than 1 401 detected, giving up", "err", err)
				return false
			}

			if res.Request.AuthToken != "" && core.ErrTokenRevoked(res.Error()) {
				slog.Warn("Token is not longer valid", "err", err)
				loginTok, loginErr := c.Login(res.Request.Context())
				if loginErr != nil {
					return false
				}
				res.Request.SetAuthToken(loginTok)
				// res.Request.SetRetryWaitTime(100 * time.Millisecond)
				// res.Request.RetryMaxWaitTime = 100 * time.Millisecond
				res.Request.Attempt = 0
				return true
			}
			return false
		}
		return true
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
