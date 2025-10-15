package c8y_api

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alarms"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/auditrecords"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/events"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/measurements"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/operations"
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

	// Username for Cumulocity Authentication
	// Username string

	// Cumulocity Tenant
	TenantName string

	// Cumulocity Version
	Version string

	// TFACode (Two Factor Authentication) code.
	TFACode string

	Cookies []*http.Cookie

	UseTenantInUsername bool

	common core.Service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the Cumulocity API.
	// Context              *ContextService
	Alarms       *alarms.Service
	AuditRecords *auditrecords.Service
	// DeviceCredentials    *DeviceCredentialsService
	Measurements *measurements.Service
	Binaries     *binaries.Service

	ManagedObjects *managedobjects.Service
	Operations     *operations.Service
	// Tenant               *TenantService
	Events *events.Service
	// Inventory            *InventoryService
	// Application          *ApplicationService
	// UIExtension          *UIExtensionService
	// ApplicationVersions  *ApplicationVersionsService
	// Identity             *IdentityService
	// Microservice         *MicroserviceService
	// Notification2        *Notification2Service
	// RemoteAccess         *RemoteAccessService
	// Retention            *RetentionRuleService
	// TenantOptions        *TenantOptionsService
	// Software             *InventorySoftwareService
	// Firmware             *InventoryFirmwareService
	// User                 *UserService
	// DeviceCertificate    *DeviceCertificateService
	// DeviceEnrollment     *DeviceEnrollmentService
	// CertificateAuthority *CertificateAuthorityService
	// Features             *FeaturesService
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

	// Not recommended
	InsecureSkipVerify bool

	Agent string
}

// NewClient returns a new Cumulocity API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(opts ClientOptions) *Client {
	circuitBreaker := resty.NewCircuitBreaker().
		SetTimeout(15 * time.Second).
		SetFailureThreshold(10).
		SetSuccessThreshold(5)

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
	SetAuth(rclient, opts.Auth)

	userAgent := defaultUserAgent
	if opts.Agent != "" {
		userAgent = opts.Agent
	}
	rclient.AddRequestMiddleware(MiddlewareAddUserAgent(userAgent, "go-client"))
	rclient.AddRequestMiddleware(MiddlewareAddHost("domain"))
	// rclient.AddRequestMiddleware(MiddlewareAuthorization(opts.Auth))

	c := &Client{
		Client:              rclient,
		BaseURL:             targetBaseURL,
		UserAgent:           userAgent,
		TenantName:          "",
		UseTenantInUsername: true,
		showSensitive:       opts.ShowSensitive,
	}
	c.common.Client = rclient
	c.Alarms = (*alarms.Service)(&c.common)
	c.AuditRecords = (*auditrecords.Service)(&c.common)
	// c.DeviceCertificate = (*DeviceCertificateService)(&c.common)
	// c.DeviceCredentials = (*DeviceCredentialsService)(&c.common)
	c.Measurements = (*measurements.Service)(&c.common)
	c.Binaries = (*binaries.Service)(&c.common)
	c.ManagedObjects = managedobjects.NewService(&c.common)
	c.Operations = (*operations.Service)(&c.common)
	// c.Tenant = (*TenantService)(&c.common)
	c.Events = events.NewService(&c.common)
	// c.Inventory = (*InventoryService)(&c.common)
	// c.DeviceEnrollment = (*DeviceEnrollmentService)(&c.common)
	// c.Application = (*ApplicationService)(&c.common)
	// c.ApplicationVersions = (*ApplicationVersionsService)(&c.common)
	// c.UIExtension = (*UIExtensionService)(&c.common)
	// c.Identity = (*IdentityService)(&c.common)
	// c.Microservice = (*MicroserviceService)(&c.common)
	// c.Notification2 = (*Notification2Service)(&c.common)
	// c.Context = (*ContextService)(&c.common)
	// c.RemoteAccess = (*RemoteAccessService)(&c.common)
	// c.Retention = (*RetentionRuleService)(&c.common)
	// c.TenantOptions = (*TenantOptionsService)(&c.common)
	// c.Software = (*InventorySoftwareService)(&c.common)
	// c.Firmware = (*InventoryFirmwareService)(&c.common)
	// c.User = (*UserService)(&c.common)
	// c.Features = (*FeaturesService)(&c.common)
	// c.CertificateAuthority = (*CertificateAuthorityService)(&c.common)
	return c
}

// SetBaseURL changes the base url used by the REST client
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
	SetAuth(c.Client, c.Auth)
}
