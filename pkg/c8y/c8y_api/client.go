package c8y_api

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/go-querystring/query"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/binaries"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/measurements"
	"resty.dev/v3"
)

var ErrNotFound = errors.New("item: not found")

var MethodsWithBody = []string{
	http.MethodDelete,
	http.MethodPatch,
	http.MethodPost,
	http.MethodPut,
}

// Check if method supports a body with the request
func RequestSupportsBody(method string) bool {
	for _, v := range MethodsWithBody {
		if strings.EqualFold(method, v) {
			return true
		}
	}
	return false
}

// ContextAuthTokenKey todo
type ContextAuthTokenKey string

// GetContextAuthTokenKey authentication key used to override the given Basic Authentication token
func GetContextAuthTokenKey() ContextAuthTokenKey {
	return ContextAuthTokenKey("authToken")
}

// ContextCommonOptionsKey todo
type ContextCommonOptionsKey string

// GetContextCommonOptionsKey common options key used to override request options for a single request
func GetContextCommonOptionsKey() ContextCommonOptionsKey {
	return ContextCommonOptionsKey("commonOptions")
}

// ContextAuthFuncKey auth function key
type ContextAuthFuncKey string

// GetContextServiceUser server user
func GetContextAuthFuncKey() ContextAuthFuncKey {
	return ContextAuthFuncKey("authFunc")
}

type AuthFunc func(r *http.Request) (string, error)

// DefaultRequestOptions default request options which are added to each outgoing request
type DefaultRequestOptions struct {
	DryRun bool

	// DryRunResponse return a mock response when using dry run
	DryRunResponse bool

	// DryRunHandler called when a request should be called
	DryRunHandler func(options *RequestOptions, req *http.Request)
}

// FromAuthFuncContext returns the AuthFunc value stored in ctx, if any.
func FromAuthFuncContext(ctx context.Context) (AuthFunc, bool) {
	u, ok := ctx.Value(GetContextAuthFuncKey()).(AuthFunc)
	return u, ok
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

	// Username for Cumulocity Authentication
	Username string

	// Cumulocity Tenant
	TenantName string

	// Cumulocity Version
	Version string

	// Password for Cumulocity Authentication
	Password string

	// Token for bearer authorization
	Token string

	// TFACode (Two Factor Authentication) code.
	TFACode string

	// Authorization type
	AuthorizationType AuthType

	Cookies []*http.Cookie

	UseTenantInUsername bool

	requestOptions DefaultRequestOptions

	common core.Service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the Cumulocity API.
	// Context              *ContextService
	// Alarm                *AlarmService
	// Audit                *AuditService
	// DeviceCredentials    *DeviceCredentialsService
	Measurement     *measurements.Service
	InventoryBinary *binaries.ManagedObjectBinaryService

	ManagedObjects *managedobjects.Service
	// Operation            *OperationService
	// Tenant               *TenantService
	// Event                *EventService
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
	defaultUserAgent = "go-client"
)

var (
	// EnvVarLoggerHideSensitive environment variable name used to control whether sensitive session information is logged or not. When set to "true", then the tenant, username, password, base 64 passwords will be obfuscated from the log messages
	EnvVarLoggerHideSensitive = "C8Y_LOGGER_HIDE_SENSITIVE"
)

const (
	// LoginTypeOAuth2Internal OAuth2 internal mode
	LoginTypeOAuth2Internal = "OAUTH2_INTERNAL"

	// LoginTypeOAuth2 OAuth2 external provider
	LoginTypeOAuth2 = "OAUTH2"

	// LoginTypeBasic Basic authentication
	LoginTypeBasic = "BASIC"

	// LoginTypeNone no authentication
	LoginTypeNone = "NONE"
)

// AuthType request authorization type
type AuthType int

const (
	// AuthTypeUnset no auth type set
	AuthTypeUnset AuthType = 0

	// AuthTypeNone don't use an Authorization
	AuthTypeNone AuthType = 1

	// AuthTypeBasic Basic Authorization
	AuthTypeBasic AuthType = 2

	// AuthTypeBearer Bearer Authorization
	AuthTypeBearer AuthType = 3
)

func (a AuthType) String() string {
	switch a {
	case AuthTypeUnset:
		return "UNSET"
	case AuthTypeNone:
		return "NONE"
	case AuthTypeBasic:
		return "BASIC"
	case AuthTypeBearer:
		return "BEARER"
	}
	return "UNKNOWN"
}

var (
	ErrInvalidLoginType = errors.New("invalid login type")
)

// Parse the login type and select as default if no value options are found
// It returns the selected method, and if the input was valid or not
func ParseLoginType(v string) (string, error) {
	v = strings.ToUpper(v)
	switch v {
	case LoginTypeBasic:
		return LoginTypeBasic, nil
	case LoginTypeNone:
		return LoginTypeNone, nil
	case LoginTypeOAuth2Internal:
		return LoginTypeOAuth2Internal, nil
	case LoginTypeOAuth2:
		return LoginTypeOAuth2, nil
	default:
		return "", ErrInvalidLoginType
	}
}

// DecodeJSONBytes decodes json preserving number formatting (especially large integers and scientific notation floats)
func DecodeJSONBytes(v []byte, dst interface{}) error {
	return DecodeJSONReader(bytes.NewReader(v), dst)
}

// DecodeJSONFile decodes a json file into dst interface
func DecodeJSONFile(filepath string, dst interface{}) error {
	fp, err := os.Open(filepath)
	if err != nil {
		return err
	}

	defer fp.Close()
	buf, err := io.ReadAll(fp)
	if err != nil {
		return err
	}
	return DecodeJSONReader(bytes.NewReader(buf), dst)
}

// DecodeJSONReader decodes bytes using a reader interface
//
// Note: Decode with the UseNumber() set so large or
// scientific notation numbers are not wrongly converted to integers!
// i.e. otherwise this conversion will happen (which causes a problem with mongodb!)
//
//	9.2233720368547758E+18 --> 9223372036854776000
func DecodeJSONReader(r io.Reader, dst interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(&dst)
}

// ClientOption represents an argument to NewClient
type ClientOption = func(http.RoundTripper) http.RoundTripper

func GetEnvValue(key ...string) string {
	for _, k := range key {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

// NewHTTPClient initializes an http.Client which can be then provided to the NewClient
func NewHTTPClient(opts ...ClientOption) *http.Client {
	tr := http.DefaultTransport
	for _, opt := range opts {
		tr = opt(tr)
	}
	return &http.Client{Transport: tr}
}

// WithClientCertificate uses the given x509 client certificate for cert-based auth when doing requests
func WithClientCertificate(cert tls.Certificate) ClientOption {
	return func(tr http.RoundTripper) http.RoundTripper {
		if tr.(*http.Transport).TLSClientConfig == nil {
			tr.(*http.Transport).TLSClientConfig = &tls.Config{
				Certificates: []tls.Certificate{cert},
			}
		} else {
			tr.(*http.Transport).TLSClientConfig.Certificates = []tls.Certificate{cert}
		}
		return tr
	}
}

func WithRequestDebugLogger(l slog.Logger) ClientOption {
	return func(tr http.RoundTripper) http.RoundTripper {
		return &LoggingTransport{
			Logger: l,
		}
	}
}

type funcTripper struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (tr funcTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return tr.roundTrip(req)
}

type LoggingTransport struct {
	Logger slog.Logger
}

func (t *LoggingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	reqInfo, _ := httputil.DumpRequestOut(r, true)
	t.Logger.Debug("Sending request.", "request", reqInfo)
	resp, err := http.DefaultTransport.RoundTrip(r)
	return resp, err
}

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

	// Username / Password Auth
	Tenant   string
	Username string
	Password string

	// Token Auth
	Token string

	// Auth preference (to control which credentials are used when more than 1 value is provided)
	AuthType AuthType

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
func NewClientV2(opts ClientOptions) *Client {
	rclient := resty.New()
	rclient.TLSClientConfig().InsecureSkipVerify = opts.InsecureSkipVerify
	rclient.SetBaseURL(opts.BaseURL)

	fmtURL := FormatBaseURL(opts.BaseURL)
	targetBaseURL, _ := url.Parse(fmtURL)

	authType := opts.AuthType
	if authType == AuthTypeUnset {
		if opts.Token != "" {
			authType = AuthTypeBearer
		} else if opts.Username != "" && opts.Password != "" {
			authType = AuthTypeBasic
		}
	}

	userAgent := defaultUserAgent
	if opts.Agent != "" {
		userAgent = opts.Agent
	}
	rclient.AddRequestMiddleware(MiddlewareAddUserAgent(userAgent, "go-client"))
	rclient.AddRequestMiddleware(MiddlewareAddHost("domain"))

	if opts.Username != "" && opts.Password != "" {
		rclient.SetBasicAuth(opts.Username, opts.Password)
	}

	c := &Client{
		Client:              rclient,
		BaseURL:             targetBaseURL,
		UserAgent:           userAgent,
		Username:            opts.Username,
		Password:            opts.Password,
		Token:               opts.Token,
		TenantName:          opts.Tenant,
		UseTenantInUsername: true,
		AuthorizationType:   authType,
		showSensitive:       opts.ShowSensitive,
	}
	c.common.Client = rclient
	// c.Alarm = (*AlarmService)(&c.common)
	// c.Audit = (*AuditService)(&c.common)
	// c.DeviceCertificate = (*DeviceCertificateService)(&c.common)
	// c.DeviceCredentials = (*DeviceCredentialsService)(&c.common)
	c.Measurement = (*measurements.Service)(&c.common)
	c.InventoryBinary = (*binaries.ManagedObjectBinaryService)(&c.common)
	c.ManagedObjects = managedobjects.NewService(&c.common)
	// c.Operation = (*OperationService)(&c.common)
	// c.Tenant = (*TenantService)(&c.common)
	// c.Event = (*EventService)(&c.common)
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

// addOptions adds the parameters in opt as URL query parameters to s. opt
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opt interface{}) (string, error) {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opt)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()

	rawQuery := u.String()
	rawQuery = rawQuery[1:]
	return rawQuery, nil
}

// Noop todo
func (c *Client) Noop() {

}

// Parse a JWT claims
func (c *Client) ParseToken(tokenString string) (*CumulocityTokenClaim, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token. expected 3 fields")
	}
	raw, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	claim := &CumulocityTokenClaim{}
	err = json.Unmarshal(raw, claim)
	return claim, err
}

// Get hostname (parse from either the token)
func (c *Client) GetHostname() string {
	if c.Token != "" {
		claims, err := c.ParseToken(c.Token)
		if err == nil {
			if c.BaseURL == nil || c.BaseURL.Host == "" {
				return claims.Issuer
			}
			if strings.Contains(c.BaseURL.Host, claims.Issuer) {
				return claims.Issuer
			}
			return c.BaseURL.Host
		}
	}
	if c.BaseURL == nil {
		return ""
	}
	return c.BaseURL.Host
}

// Get the username. Parse the token if exists
func (c *Client) GetUsername() string {
	if c.Token != "" {
		claims, err := c.ParseToken(c.Token)
		if err == nil {
			return claims.User
		}
	}
	return c.Username
}

// NewAuthorizationContextFromRequest returns a new context with the Authorization token set which will override the Basic Auth in subsequent
// REST requests
func NewAuthorizationContextFromRequest(req *http.Request) context.Context {
	if req == nil {
		return context.Background()
	}
	auth := req.Header.Get("Authorization")
	return context.WithValue(context.Background(), GetContextAuthTokenKey(), auth)
}

// NewAuthorizationContext returns context with the Authorization token set given explicit tenant, username and password.
func NewAuthorizationContext(tenant, username, password string) context.Context {
	auth := NewBasicAuthString(tenant, username, password)
	return context.WithValue(context.Background(), GetContextAuthTokenKey(), auth)
}

// NewBasicAuthAuthorizationContext returns a new basic authorization context
func NewBasicAuthAuthorizationContext(ctx context.Context, tenant, username, password string) context.Context {
	auth := NewBasicAuthString(tenant, username, password)
	return context.WithValue(ctx, GetContextAuthTokenKey(), auth)
}

// NewBearerAuthAuthorizationContext returns a new bearer authorization context
func NewBearerAuthAuthorizationContext(ctx context.Context, token string) context.Context {
	auth := NewBearerAuthString(token)
	return context.WithValue(ctx, GetContextAuthTokenKey(), auth)
}

// NewBasicAuthString returns a Basic Authorization key used for rest requests
func NewBasicAuthString(tenant, username, password string) string {
	var auth string
	if tenant == "" {
		auth = fmt.Sprintf("%s:%s", username, password)
	} else {
		auth = fmt.Sprintf("%s/%s:%s", tenant, username, password)
	}

	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// NewBearerAuthString returns a Bearer Authorization key used for rest requests
func NewBearerAuthString(token string) string {
	return "Bearer " + token
}

// Request validator function to be used to check if the outgoing request is properly formulated
type RequestValidator func(*http.Request) error

// RequestOptions struct which contains the options to be used with the SendRequest function
type RequestOptions struct {
	Method         string
	Host           string
	Path           string
	Accept         string
	ContentType    string
	Query          interface{} // Use string if you want
	Body           interface{}
	ResponseData   interface{}
	FormData       []*resty.MultipartField
	Header         http.Header
	IgnoreAccept   bool
	AuthFunc       RequestAuthFunc
	DryRun         bool
	DryRunResponse bool
	ValidateFuncs  []RequestValidator
	PrepareRequest func(*http.Request) (*http.Request, error)

	PrepareRequestOnDryRun bool
}

// Add a validator function which will check if the outgoing http request is valid or not
func (r *RequestOptions) WithValidateFunc(v ...RequestValidator) *RequestOptions {
	if r.ValidateFuncs == nil {
		r.ValidateFuncs = make([]RequestValidator, 0)
	}
	r.ValidateFuncs = append(r.ValidateFuncs, v...)
	return r
}

func (r *RequestOptions) GetPath() (string, error) {
	prefixPath := ""
	if r.Host != "" {
		if u, err := url.Parse(r.Host); err == nil {
			prefixPath = u.Path
		}
	}

	tempURL, err := url.Parse(r.Path)
	if err != nil {
		return "", err
	}

	tempURL.Path = path.Join(prefixPath, tempURL.Path)
	return tempURL.Path, nil
}

func (r *RequestOptions) GetEscapedPath() (string, error) {
	p, err := r.GetPath()
	if err != nil {
		return "", err
	}
	return strings.ReplaceAll(url.PathEscape(p), "%2F", "/"), nil
}

func (r *RequestOptions) GetQuery() (string, error) {
	tempURL, err := url.Parse(r.Path)
	if err != nil {
		return "", err
	}

	queryParams := tempURL.Query()

	if r.Query != nil {
		queryPart, ok := r.Query.(string)
		if !ok {
			if v, err := addOptions("", r.Query); err == nil {
				queryPart = v
			} else {
				return "", err
			}
		}

		if queryPart != "" {
			query, _ := url.ParseQuery(queryPart)

			for key, query := range query {
				for _, qValue := range query {
					queryParams.Add(key, qValue)
				}
			}
		}
	}

	return queryParams.Encode(), nil
	// return queryParams.Encode(), nil
}

func (r *RequestOptions) GetQueryParameterValues() (url.Values, error) {
	tempURL, err := url.Parse(r.Path)
	if err != nil {
		return nil, err
	}

	queryParams := tempURL.Query()

	if r.Query != nil {
		queryPart, ok := r.Query.(string)
		if !ok {
			if v, err := addOptions("", r.Query); err == nil {
				queryPart = v
			} else {
				return queryParams, err
			}
		}

		if queryPart != "" {
			query, _ := url.ParseQuery(queryPart)

			for key, query := range query {
				for _, qValue := range query {
					queryParams.Add(key, qValue)
				}
			}
		}
	}

	return queryParams, nil
}

// ensureRelativePath returns a relative path variant of the input path.
// e.g. /test/path => test/path
func ensureRelativePath(u string) string {
	return strings.TrimPrefix(u, "/")
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

func (c *Client) SetHostHeader(req *resty.Request) {
	if req != nil && c.Domain != "" && c.Domain != req.RawRequest.URL.Host {
		// setting the Host header actually does nothing however
		// it makes the setting visible when logging
		req.Header.Set("Host", c.Domain)
		req.RawRequest.Host = c.Domain
	}
}

// SetAuthorization sets the configured authorization to the given request. By default it will set the Basic Authorization header
func (c *Client) SetAuthorization(req *resty.Request, authTypeFunc ...RequestAuthFunc) (bool, error) {
	if len(authTypeFunc) > 0 {
		return authTypeFunc[0](req)
	}

	authType := c.AuthorizationType
	if authType == AuthTypeUnset {
		if c.Token != "" {
			authType = AuthTypeBearer
		} else if c.Username != "" && c.Password != "" {
			authType = AuthTypeBasic
		}
	}

	switch authType {
	case AuthTypeNone:
		return WithNoAuthorization()(req)
	case AuthTypeBearer:
		c.addCookiesToRequest(req)
		return WithToken(c.Token)(req)
	case AuthTypeBasic:
		if c.UseTenantInUsername {
			return WithTenantUsernamePassword(c.TenantName, c.Username, c.Password)(req)
		} else {
			return WithTenantUsernamePassword("", c.Username, c.Password)(req)
		}
	}
	return false, nil
}

// GetXSRFToken returns the XSRF Token if found in the configured cookies
func (c *Client) GetXSRFToken() string {
	for _, cookie := range c.Cookies {
		if strings.ToUpper(cookie.Name) == "XSRF-TOKEN" {
			return cookie.Value
		}
	}
	return ""
}

// SetCookies set the cookies to use for all rest requests
func (c *Client) SetCookies(cookies []*http.Cookie) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.Cookies = cookies
}

// SetAuthorizationType set the authorization type to use to add to outgoing requests
func (c *Client) SetAuthorizationType(authType AuthType) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.AuthorizationType = authType
}

// SetTenantUsernamePassword sets the tenant/username/password to use for all rest requests
func (c *Client) SetTenantUsernamePassword(tenant string, username string, password string) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.TenantName = tenant
	c.Username = username
	c.Password = password
	c.AuthorizationType = AuthTypeBasic
}

// SetUsernamePassword sets the username/password to use for all rest requests
// If the username is in the format of {tenant}/{username}, then the tenant
// name will also be set with the given value
func (c *Client) SetUsernamePassword(username string, password string) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()

	if tenant, user, found := strings.Cut(username, "/"); found {
		username = user
		c.TenantName = tenant
		c.UseTenantInUsername = true
	}

	c.Username = username
	c.Password = password
	c.AuthorizationType = AuthTypeBasic
}

// ClearTenantUsernamePassword removes any tenant, username and password set on the client
func (c *Client) ClearTenantUsernamePassword() {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.TenantName = ""
	c.Username = ""
	c.Password = ""
}

// SetToken sets the Bearer auth token to use for all rest requests
func (c *Client) SetToken(v string) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.Token = v
	c.AuthorizationType = AuthTypeBearer
}

// Clear an existing token
func (c *Client) ClearToken() {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.Token = ""
}

func (c *Client) addCookiesToRequest(req *resty.Request) {
	if c.Cookies == nil {
		return
	}

	cookieValues := make([]string, 0)
	for _, cookie := range c.Cookies {
		if cookie.Name == "XSRF-TOKEN" {
			req.Header.Set("X-"+cookie.Name, cookie.Value)
		} else {
			cookieValues = append(cookieValues, fmt.Sprintf("%s=%s", cookie.Name, cookie.Value))
		}
	}

	if len(cookieValues) > 0 {
		req.Header.Add("Cookie", strings.Join(cookieValues, "; "))
	}
}

// OAuthTokenResponse OAuth Token Response
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token,omitempty"`
}

func withContext(ctx context.Context, req *http.Request) *http.Request {
	return req.WithContext(ctx)
}

func toKB(v int64) string {
	if v == -1 {
		return "-1"
	}
	return fmt.Sprintf("%0.1fKB", float64(v/1024))
}

// sanitizeURL redacts the client_secret parameter from the URL which may be
// exposed to the user.
func sanitizeURL(uri *url.URL) *url.URL {
	if uri == nil {
		return nil
	}
	params := uri.Query()
	if len(params.Get("client_secret")) > 0 {
		params.Set("client_secret", "REDACTED")
		uri.RawQuery = params.Encode()
	}
	return uri
}

/*
An Error reports more details on an individual error in an ErrorResponse.
These are the possible validation error codes:

	missing:
	    resource does not exist
	missing_field:
	    a required field on a resource has not been set
	invalid:
	    the formatting of a field is invalid
	already_exists:
	    another resource has the same valid as this field
	custom:
	    some resources return this (e.g. github.User.CreateKey()), additional
	    information is set in the Message field of the Error
*/
type Error struct {
	Resource     string `json:"resource"` // resource on which the error occurred
	Field        string `json:"field"`    // field on which the error occurred
	Code         string `json:"code"`     // validation error code
	Message      string `json:"message"`  // Message describing the error. Errors with Code == "custom" will always have this set.
	ErrorMessage string `json:"error"`
	Information  string `json:"info"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v error caused by %v field on %v resource",
		e.Code, e.Field, e.Resource)
}

/*
An ErrorResponse reports one or more errors caused by an API request.
*/
type ErrorResponse struct {
	RawResponse *http.Response
	ErrorType   string `json:"error,omitempty"`   // Error type formatted as "<<resource type>>/<<error name>>"". For example, an object not found in the inventory is reported as "inventory/notFound".
	Message     string `json:"message,omitempty"` // error message
	Info        string `json:"info,omitempty"`    // URL to an error description on the Internet.

	// Error details. Only available in DEBUG mode.
	Details *struct {
		ExceptionClass      string `json:"exceptionClass,omitempty"`
		ExceptionMessage    string `json:"exceptionMessage,omitempty"`
		ExceptionStackTrace string `json:"exceptionStackTrace,omitempty"`
	} `json:"details,omitempty"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v %v",
		r.RawResponse.Request.Method, sanitizeURL(r.RawResponse.Request.URL),
		r.RawResponse.StatusCode, r.ErrorType, r.Message)
}

// DefaultDryRunHandler is the default dry run handler
func (c *Client) DefaultDryRunHandler(options *RequestOptions, req *http.Request) {
	// Show information about the request i.e. url, headers, body etc.
	message := fmt.Sprintf("What If: Sending [%s] request to [%s]\n", req.Method, req.URL)

	if len(req.Header) > 0 {
		message += "\nHeaders:\n"
	}

	// sort header names
	headerNames := make([]string, 0, len(req.Header))
	for key := range req.Header {
		headerNames = append(headerNames, key)
	}

	sort.Strings(headerNames)

	for _, key := range headerNames {
		val := req.Header[key]
		message += fmt.Sprintf("%s: %s\n", key, val[0])
	}

	if options.Body != nil && RequestSupportsBody(req.Method) {
		if v, parseErr := json.MarshalIndent(options.Body, "", "  "); parseErr == nil && !bytes.Equal(v, []byte("null")) {
			message += fmt.Sprintf("\nBody:\n%s", v)
		} else {
			// TODO: check if this can display body reader as string?
			message += fmt.Sprintf("\nBody:\n%v", options.Body)
		}
	} else {
		message += "\nBody: (empty)\n"
	}

	if len(options.FormData) > 0 {
		message += "\nForm Data:\n"
		for _, item := range options.FormData {
			if item.Name == "file" {
				message += fmt.Sprintf("%s: (file contents)\n", item.Name)
			} else {
				buf := new(strings.Builder)
				if _, err := io.Copy(buf, item.Reader); err == nil {
					message += fmt.Sprintf("%s: %s\n", item.Name, buf.String())
				}
			}
		}
	}

	slog.Info(message)
}

type CumulocityTokenClaim struct {
	User      string `json:"sub,omitempty"`
	Tenant    string `json:"ten,omitempty"`
	XSRFToken string `json:"xsrfToken,omitempty"`
	TGA       bool   `json:"tfa,omitempty"`
	jwt.RegisteredClaims
}

// Token claims
// ------------
// {
//   "aud": "test-ci-runner01.latest.stage.c8y.io",
//   "exp": 1688664540,
//   "iat": 1687454940,
//   "iss": "test-ci-runner01.latest.stage.c8y.io",
//   "jti": "0b912809-9782-4f80-b81f-50616b9aea7f",
//   "nbf": 1687454940,
//   "sub": "ciuser01",
//   "tci": "e92245a3-f088-4490-bda7-54027ba31af5",
//   "ten": "t2873877",
//   "tfa": false,
//   "xsrfToken": "UTpiVqeHmaCHAedigjZS"
// }
