package c8y

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/go-querystring/query"
	"github.com/tidwall/gjson"
	"moul.io/http2curl"
)

// ContextAuthTokenKey todo
type ContextAuthTokenKey string

// GetContextAuthTokenKey authentication key used to override the given Basic Authentication token
func GetContextAuthTokenKey() ContextAuthTokenKey {
	return ContextAuthTokenKey("authToken")
}

// DefaultRequestOptions default request options which are added to each outgoing request
type DefaultRequestOptions struct {
	DryRun bool
}

type service struct {
	client *Client
}

// A Client manages communication with the Cumulocity API.
type Client struct {
	clientMu sync.Mutex   // clientMu protects the client during calls that modify the CheckRedirect func.
	client   *http.Client // HTTP client used to communicate with the API.

	Realtime *RealtimeClient

	// Base URL for API requests. Defaults to the public Cumulocity API, but can be
	// set to a domain endpoint to use with Cumulocity. BaseURL should
	// always be specified with a trailing slash.
	BaseURL *url.URL

	// User agent used when communicating with the Cumulocity API.
	UserAgent string

	// Username for Cumulocity Authentication
	Username string

	// Cumulocity Tenant
	TenantName string

	// Password for Cumulocity Authentication
	Password string

	// TFACode (Two Factor Authentication) code.
	TFACode string

	// Authorization method
	AuthorizationMethod string

	Cookies []*http.Cookie

	UseTenantInUsername bool

	requestOptions DefaultRequestOptions

	// Microservice bootstrap and service users
	BootstrapUser ServiceUser
	ServiceUsers  []ServiceUser

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the Cumulocity API.
	Context           *ContextService
	Alarm             *AlarmService
	Audit             *AuditService
	DeviceCredentials *DeviceCredentialsService
	Measurement       *MeasurementService
	Operation         *OperationService
	Tenant            *TenantService
	Event             *EventService
	Inventory         *InventoryService
	Application       *ApplicationService
	Identity          *IdentityService
	Microservice      *MicroserviceService
	Retention         *RetentionRuleService
	TenantOptions     *TenantOptionsService
	User              *UserService
}

const (
	defaultUserAgent = "go-client"
)

var (
	// EnvVarLoggerHideSensitive environment variable name used to control whethere sensitive session information is logged or not. When set to "true", then the tenant, username, password, base 64 passwords will be obfuscated from the log messages
	EnvVarLoggerHideSensitive = "C8Y_LOGGER_HIDE_SENSITIVE"
)

const (
	// AuthMethodOAuth2Internal OAuth2 internal mode
	AuthMethodOAuth2Internal = "OAUTH2_INTERNAL"

	// AuthMethodBasic Basic authentication
	AuthMethodBasic = "BASIC"

	// AuthMethodNone no authentication
	AuthMethodNone = "NONE"
)

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
	buf, err := ioutil.ReadAll(fp)
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
//  	9.2233720368547758E+18 --> 9223372036854776000
//
func DecodeJSONReader(r io.Reader, dst interface{}) error {
	decoder := json.NewDecoder(r)
	decoder.UseNumber()
	return decoder.Decode(&dst)
}

// NewRealtimeClientFromServiceUser returns a realtime client using a microservice's service user for a specified tenant
// If no service user is found for the set tenant, then nil is returned
func (c *Client) NewRealtimeClientFromServiceUser(tenant string) *RealtimeClient {
	if len(c.ServiceUsers) == 0 {
		Logger.Fatal("No service users found")
	}
	for _, user := range c.ServiceUsers {
		if tenant == user.Tenant || tenant == "" {
			return NewRealtimeClient(c.BaseURL.String(), nil, user.Tenant, user.Username, user.Password)
		}
	}
	return nil
}

// NewClientFromEnvironment returns a new c8y client configured from environment variables
//
// Environment Variables
// C8Y_HOST - Cumulocity host server address e.g. https://cumulocity.com
// C8Y_TENANT - Tenant name e.g. mycompany
// C8Y_USER - Username e.g. myuser@mycompany.com
// C8Y_PASSWORD - Password
//
func NewClientFromEnvironment(httpClient *http.Client, skipRealtimeClient bool) *Client {
	baseURL := os.Getenv("C8Y_HOST")
	tenant, username, password := GetServiceUserFromEnvironment()
	return NewClient(httpClient, baseURL, tenant, username, password, skipRealtimeClient)
}

// NewClientUsingBootstrapUserFromEnvironment returns a Cumulocity client using the the bootstrap credentials set in the environment variables
func NewClientUsingBootstrapUserFromEnvironment(httpClient *http.Client, baseURL string, skipRealtimeClient bool) *Client {
	tenant, username, password := GetBootstrapUserFromEnvironment()

	client := NewClient(httpClient, baseURL, tenant, username, password, skipRealtimeClient)
	client.Microservice.SetServiceUsers()
	return client
}

// NewClient returns a new Cumulocity API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(httpClient *http.Client, baseURL string, tenant string, username string, password string, skipRealtimeClient bool) *Client {
	if httpClient == nil {
		// Default client ignores self signed certificates (to enable compatibility to the edge which uses self signed certs)
		defaultTransport := http.DefaultTransport.(*http.Transport)
		tr := &http.Transport{
			Proxy:                 defaultTransport.Proxy,
			DialContext:           defaultTransport.DialContext,
			MaxIdleConns:          defaultTransport.MaxIdleConns,
			IdleConnTimeout:       defaultTransport.IdleConnTimeout,
			ExpectContinueTimeout: defaultTransport.ExpectContinueTimeout,
			TLSHandshakeTimeout:   defaultTransport.TLSHandshakeTimeout,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}

		httpClient = &http.Client{
			Transport: tr,
		}
	}

	var fmtURL string
	if !strings.HasSuffix(baseURL, "/") {
		fmtURL = baseURL + "/"
	} else {
		fmtURL = baseURL
	}
	targetBaseURL, _ := url.Parse(fmtURL)

	var realtimeClient *RealtimeClient
	if !skipRealtimeClient {
		Logger.Infof("Creating realtime client %s", fmtURL)
		realtimeClient = NewRealtimeClient(fmtURL, nil, tenant, username, password)
	}

	userAgent := defaultUserAgent

	c := &Client{
		client:              httpClient,
		BaseURL:             targetBaseURL,
		UserAgent:           userAgent,
		Realtime:            realtimeClient,
		Username:            username,
		Password:            password,
		TenantName:          tenant,
		UseTenantInUsername: true,
	}
	c.common.client = c
	c.Alarm = (*AlarmService)(&c.common)
	c.Audit = (*AuditService)(&c.common)
	c.DeviceCredentials = (*DeviceCredentialsService)(&c.common)
	c.Measurement = (*MeasurementService)(&c.common)
	c.Operation = (*OperationService)(&c.common)
	c.Tenant = (*TenantService)(&c.common)
	c.Event = (*EventService)(&c.common)
	c.Inventory = (*InventoryService)(&c.common)
	c.Application = (*ApplicationService)(&c.common)
	c.Identity = (*IdentityService)(&c.common)
	c.Microservice = (*MicroserviceService)(&c.common)
	c.Context = (*ContextService)(&c.common)
	c.Retention = (*RetentionRuleService)(&c.common)
	c.TenantOptions = (*TenantOptionsService)(&c.common)
	c.User = (*UserService)(&c.common)
	return c
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

// NewBasicAuthString returns a Basic Authorization key used for rest requests
func NewBasicAuthString(tenant, username, password string) string {
	auth := fmt.Sprintf("%s/%s:%s", tenant, username, password)
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))
}

// RequestOptions struct which contains the options to be used with the SendRequest function
type RequestOptions struct {
	Method           string
	Host             string
	Path             string
	Accept           string
	ContentType      string
	Query            interface{} // Use string if you want
	Body             interface{}
	ResponseData     interface{}
	FormData         map[string]io.Reader
	Header           http.Header
	IgnoreAccept     bool
	NoAuthentication bool
	DryRun           bool
	DryRunResponse   bool
}

// SendRequest creates and sends a request
func (c *Client) SendRequest(ctx context.Context, options RequestOptions) (*Response, error) {

	localLogger := Logger
	// if options.Logger == nil {
	// 	localLogger = *options.Logger
	// }

	queryParams := ""

	if options.Query != nil {
		if v, ok := options.Query.(string); ok {
			queryParams = v
		} else {
			if v, err := addOptions("", options.Query); err == nil {
				queryParams = v
			} else {
				localLogger.Infof("ERROR: Could not convert query parameter interface{} to a string. %s", err)
				return nil, err
			}
		}
	}

	var req *http.Request
	var err error

	if len(options.FormData) > 0 {
		localLogger.Infof("Sending multipart form-data")
		// Process FormData (for multipart/form-data requests)
		// TODO: Somehow use the c.NewRequest function as it provides
		// the authentication required for the request
		u, _ := url.Parse(c.BaseURL.String())
		u.Path = path.Join(u.Path, options.Path)
		req, err = prepareMultipartRequest(options.Method, u.String(), options.FormData)
		if !options.NoAuthentication {
			c.SetAuthorization(req)
		}
	} else {
		// Normal request
		if options.NoAuthentication {
			req, err = c.NewRequestWithoutAuth(options.Method, options.Path, queryParams, options.Body)
		} else {
			req, err = c.NewRequest(options.Method, options.Path, queryParams, options.Body)
		}
	}

	if err != nil {
		return nil, err
	}

	if !options.IgnoreAccept {
		if req.Header.Get("Accept") == "" {
			acceptType := "application/json"
			if options.Accept != "" {
				acceptType = options.Accept
			}
			req.Header.Set("Accept", acceptType)
		}
	} else {
		req.Header.Del("Accept")
	}

	if options.ContentType != "" {
		req.Header.Set("Content-Type", options.ContentType)
	}

	if options.Header != nil {
		for name, values := range options.Header {
			// Delete any existing header
			req.Header.Del(name)

			// Transfer the values
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}
	}

	if options.Host != "" {
		host := options.Host
		if !strings.HasPrefix(options.Host, "https://") && !strings.HasPrefix(options.Host, "http://") {
			host = "https://" + options.Host
		}
		baseURL, parseErr := url.Parse(host)

		if parseErr != nil {
			localLogger.Warnf("Ignoring invalid host %s. %s", host, parseErr)
			err = parseErr
		} else {
			req.URL.Host = baseURL.Host
			req.URL.Scheme = baseURL.Scheme
			localLogger.Infof("Using alternative host %s://%s", req.URL.Scheme, req.URL.Host)
		}

	}

	if err != nil {
		return nil, err
	}

	if options.DryRun || c.requestOptions.DryRun {
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

		if options.Body != nil && (req.Method == http.MethodPost || req.Method == http.MethodPut || req.Method == http.MethodPatch) {
			if v, parseErr := json.MarshalIndent(options.Body, "", "  "); parseErr == nil && !bytes.Equal(v, []byte("null")) {
				message += fmt.Sprintf("\nBody:\n%s", v)
			} else {
				// TODO: check if this can display body reader as string?
				message += fmt.Sprintf("\nBody:\n%v", options.Body)
			}
		} else {
			message += "\nBody: (empty)\n"
		}

		if options.FormData != nil && len(options.FormData) > 0 {
			message += "\nForm Data:\n"
			for key, formData := range options.FormData {
				if key == "file" {
					message += fmt.Sprintf("%s: (file contents)\n", key)
				} else {
					buf := new(strings.Builder)
					if _, err := io.Copy(buf, formData); err == nil {
						message += fmt.Sprintf("%s: %s\n", key, buf.String())
					}
				}
			}
		}

		localLogger.Info(c.hideSensitiveInformationIfActive(message))

		if command, curlErr := http2curl.GetCurlCommand(req); curlErr == nil {
			_ = command
			// localLogger.Infof("curl: %s", strings.ReplaceAll(command.String(), "\"", "\\\""))
		}

		if options.DryRunResponse {
			return &Response{
				Response: &http.Response{
					Request: req,
				},
			}, nil
		}
		return nil, nil
	}

	localLogger.Info(c.hideSensitiveInformationIfActive(fmt.Sprintf("Headers: %v", req.Header)))

	resp, err := c.Do(ctx, req, options.ResponseData)

	c.SetJSONItems(resp, options.ResponseData)

	if err != nil {
		return resp, err
	}
	return resp, nil
}

// SetJSONItems sets the GJSON items to the input v object
func (c *Client) SetJSONItems(resp *Response, v interface{}) error {
	if resp == nil {
		return nil
	}
	// data.Item = gjson.Parse(resp.JSON.Raw)

	switch t := v.(type) {
	case *Alarm:
		t.Item = *resp.JSON
	case *AlarmCollection:
		t.Items = resp.JSON.Get("alarms").Array()

	case *Application:
		t.Item = *resp.JSON
	case *ApplicationCollection:
		t.Items = resp.JSON.Get("applications").Array()

	case *AuditRecord:
		t.Item = *resp.JSON
	case *AuditRecordCollection:
		t.Items = resp.JSON.Get("auditRecords").Array()

	case *Event:
		t.Item = *resp.JSON
	case *EventCollection:
		t.Items = resp.JSON.Get("events").Array()

	case *EventBinary:
		t.Item = *resp.JSON

	case *GroupCollection:
		t.Items = resp.JSON.Get("groups").Array()

	case *Identity:
		t.Item = *resp.JSON

	case *ManagedObject:
		t.Item = *resp.JSON
	case *ManagedObjectCollection:
		t.Items = resp.JSON.Get("managedObjects").Array()

	case *Measurement:
		t.Item = *resp.JSON
	case *Measurements:
		t.Items = resp.JSON.Get("measurements").Array()
	case *MeasurementCollection:
		t.Items = resp.JSON.Get("measurements").Array()

	case *Operation:
		t.Item = *resp.JSON
	case *OperationCollection:
		t.Items = resp.JSON.Get("operations").Array()

	case *RoleCollection:
		t.Items = resp.JSON.Get("roles").Array()

	case *TenantOption:
		t.Item = *resp.JSON
	case *TenantOptionCollection:
		t.Items = resp.JSON.Get("options").Array()

	case *UserCollection:
		t.Items = resp.JSON.Get("users").Array()

	}

	return nil
}

// NewRequest returns a request with the required additional base url, authentication header, accept and user-agent.NewRequest
func (c *Client) NewRequest(method, path string, query string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}

	rel := &url.URL{Path: path}
	if query != "" {
		rel.RawQuery = query
	}

	u := c.BaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		switch v := body.(type) {
		case *os.File:
			buf = v
		case string:
			buf = bytes.NewBufferString(v)
		case []byte:
			buf = bytes.NewBuffer(v)
		case io.ReadWriter:
			buf = v
		default:
			buf = new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(body)

			if err != nil {
				return nil, err
			}
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		if strings.ToUpper(method) != "GET" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	c.SetAuthorization(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-APPLICATION", "go-client")
	return req, nil
}

// NewRequestWithoutAuth returns a request with the required additional base url, accept and user-agent.NewRequest
func (c *Client) NewRequestWithoutAuth(method, path string, query string, body interface{}) (*http.Request, error) {
	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}

	rel := &url.URL{Path: path}
	if query != "" {
		rel.RawQuery = query
	}

	u := c.BaseURL.ResolveReference(rel)

	var buf io.ReadWriter
	if body != nil {
		switch v := body.(type) {
		case *os.File:
			buf = v
		case string:
			buf = bytes.NewBufferString(v)
		case []byte:
			buf = bytes.NewBuffer(v)
		case io.ReadWriter:
			buf = v
		default:
			buf = new(bytes.Buffer)
			err := json.NewEncoder(buf).Encode(body)

			if err != nil {
				return nil, err
			}
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		if strings.ToUpper(method) != "GET" {
			req.Header.Set("Content-Type", "application/json")
		}
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-APPLICATION", "go-client")
	return req, nil
}

// SetBasicAuthorization sets the configured authorization to the given request. By default it will set the Basic Authorization header
func (c *Client) SetBasicAuthorization(req *http.Request) {
	var headerUsername string
	if c.UseTenantInUsername && c.TenantName != "" {
		headerUsername = fmt.Sprintf("%s/%s", c.TenantName, c.Username)
	} else {
		headerUsername = c.Username
	}
	Logger.Infof("Current username: %s", c.hideSensitiveInformationIfActive(headerUsername))
	req.SetBasicAuth(headerUsername, c.Password)
}

// SetAuthorization sets the configured authorization to the given request. By default it will set the Basic Authorization header
func (c *Client) SetAuthorization(req *http.Request) {
	switch c.AuthorizationMethod {
	case AuthMethodOAuth2Internal:
		c.addOAuth2ToRequest(req)
	case AuthMethodNone:
		break
	case AuthMethodBasic:
		fallthrough
	default:
		c.SetBasicAuthorization(req)
	}
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

func (c *Client) addOAuth2ToRequest(req *http.Request) {
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

// LoginUsingOAuth2 login to Cumulocity using OAuth2 and save the cookies from the response
func (c *Client) LoginUsingOAuth2(ctx context.Context, initRequest ...string) error {

	data := url.Values{}
	data.Set("grant_type", "PASSWORD")
	data.Set("username", c.Username)
	data.Set("password", c.Password)
	tfaCode := "undefined"
	if c.TFACode != "" {
		tfaCode = c.TFACode
	}
	data.Set("tfa_code", tfaCode)

	headers := http.Header{}
	headers.Add("Content-Length", strconv.Itoa(len(data.Encode())))

	options := &RequestOptions{
		Method:      "POST",
		Path:        "/tenant/oauth",
		Query:       fmt.Sprintf("tenant_id=%s", c.TenantName),
		Accept:      "*/*",
		Header:      headers,
		ContentType: "application/x-www-form-urlencoded;charset=UTF-8",
		Body:        data.Encode(),
	}

	if len(initRequest) > 0 && initRequest[0] != "" {
		if v, err := url.Parse(initRequest[0]); err == nil {
			options.Path = v.Path
			options.Query = v.RawQuery
		}
	}

	resp, err := c.SendRequest(
		ctx,
		*options,
	)

	if err != nil {
		return err
	}

	c.SetCookies(resp.Cookies())

	// test
	c.AuthorizationMethod = AuthMethodOAuth2Internal
	tenant, _, err := c.Tenant.GetCurrentTenant(ctx)
	if err != nil {
		return err
	}
	c.TenantName = tenant.Name
	return nil
}

// SetRequestOptions sets default request options to use in all requests
func (c *Client) SetRequestOptions(options DefaultRequestOptions) {
	c.clientMu.Lock()
	defer c.clientMu.Unlock()
	c.requestOptions = options
}

// Response is a Cumulocity API response. This wraps the standard http.Response
// returned from Cumulocity and provides convenient access to things like
// pagination links.
type Response struct {
	*http.Response

	// JSONData raw json response
	JSONData *string

	// JSON
	JSON *gjson.Result
}

// DecodeJSON returns the json response decoded into the given interface
func (r *Response) DecodeJSON(v interface{}) error {
	if r.JSON == nil {
		return fmt.Errorf("JSON object does not exist (i.e. is nil)")
	}
	err := DecodeJSONBytes([]byte(r.JSON.Raw), v)

	if err != nil {
		return err
	}
	return nil
}

// newResponse creates a new Response for the provided http.Response.
// r must not be nil.
func newResponse(r *http.Response) *Response {
	response := &Response{Response: r}

	// Copy the r.Body into another reader, so it is left "untouched"
	// https://stackoverflow.com/questions/23070876/reading-body-of-http-request-without-modifying-request-state
	buf, _ := ioutil.ReadAll(r.Body)
	rdr1 := ioutil.NopCloser(bytes.NewBuffer(buf))
	rdr2 := ioutil.NopCloser(bytes.NewBuffer(buf))
	bodyBytes, _ := ioutil.ReadAll(rdr1)
	bodyString := string(bodyBytes)
	response.JSONData = &bodyString

	jsonObject := gjson.Parse(bodyString)
	response.JSON = &jsonObject

	r.Body = rdr2
	return response
}

func withContext(ctx context.Context, req *http.Request) *http.Request {
	return req.WithContext(ctx)
}

// Do sends an API request and returns the API response. The API response is
// JSON decoded and stored in the value pointed to by v, or returned as an
// error if an API error has occurred. If v implements the io.Writer
// interface, the raw response body will be written to v, without attempting to
// first decode it.
//
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = withContext(ctx, req)

	// Check if an authorization key is provided in the context, if so then override the c8y authentication
	if authToken := ctx.Value(GetContextAuthTokenKey()); authToken != nil {
		Logger.Infof("Overriding basic auth provided in the context")
		req.Header.Set("Authorization", authToken.(string))
	}

	if req != nil {
		Logger.Infof("Sending request: %s %s", req.Method, c.hideSensitiveInformationIfActive(req.URL.String()))
	}

	// Log the body (if applicable)
	if req != nil && req.Body != nil {
		switch v := req.Body.(type) {
		case *os.File:
			// Only log the file name
			Logger.Infof("Body (file): %s", v.Name())
		default:
			// Don't print out multi part forms, but everything else is fine.
			if !strings.Contains(req.Header.Get("Content-Type"), "multipart/form-data") {
				// bodyBytes, _ := ioutil.ReadAll(io.LimitReader(v, 4096))
				bodyBytes, _ := ioutil.ReadAll(v)
				req.Body.Close() //  must close
				req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
				Logger.Infof("Body: %s", bytes.TrimSpace(bodyBytes))
			}
		}
	}

	resp, err := c.client.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		Logger.Infof("ERROR: Request failed. %s", err)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// If the error type is *url.Error, sanitize its URL before returning.
		if e, ok := err.(*url.Error); ok {
			if url, parseErr := url.Parse(e.URL); parseErr == nil {
				e.URL = sanitizeURL(url).String()
				return nil, e
			}
		}

		return nil, err
	}

	defer resp.Body.Close()

	response := newResponse(resp)

	err = CheckResponse(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		Logger.Infof("Invalid response received from server. %s", err)
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			err = DecodeJSONReader(resp.Body, v)

			if err == io.EOF {
				Logger.Infof("Error decoding body. %s", err)
				err = nil // ignore EOF errors caused by empty response body
			}
		}
	}

	Logger.Info(fmt.Sprintf("Status code: %v", response.StatusCode))

	return response, err
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
	Response  *http.Response `json:"-"`                 // HTTP response that caused this error
	ErrorType string         `json:"error,omitempty"`   // Error type formatted as "<<resource type>>/<<error name>>"". For example, an object not found in the inventory is reported as "inventory/notFound".
	Message   string         `json:"message,omitempty"` // error message
	Info      string         `json:"info,omitempty"`    // URL to an error description on the Internet.

	// Error details. Only available in DEBUG mode.
	Details *struct {
		ExpectionClass      string `json:"expectionClass,omitempty"`
		ExceptionMessage    string `json:"exceptionMessage,omitempty"`
		ExpectionStackTrace string `json:"expectionStackTrace,omitempty"`
	} `json:"details,omitempty"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v %v",
		r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
		r.Response.StatusCode, r.ErrorType, r.Message)
}

// AcceptedError occurs when Cumulocity returns 202 Accepted response with an
// empty body, which means a job was scheduled on the Cumulocity side to process
// the information needed and cache it.
// Technically, 202 Accepted is not a real error, it's just used to
// indicate that results are not ready yet, but should be available soon.
// The request can be repeated after some time.
type AcceptedError struct{}

func (*AcceptedError) Error() string {
	return "job scheduled on Cumulocity side; try again later"
}

// CheckResponse checks the API response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range or equal to 202 Accepted.
// API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other
// response body will be silently ignored.
//
func CheckResponse(r *http.Response) error {
	if r.StatusCode == http.StatusAccepted {
		return &AcceptedError{}
	}
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)

	if err == nil && data != nil {
		DecodeJSONBytes(data, errorResponse)
	}
	return errorResponse
}

func (c *Client) hideSensitiveInformationIfActive(message string) string {

	if strings.ToLower(os.Getenv(EnvVarLoggerHideSensitive)) != "true" {
		return message
	}

	if os.Getenv("USERNAME") != "" {
		message = strings.ReplaceAll(message, os.Getenv("USERNAME"), "******")
	}
	message = strings.ReplaceAll(message, c.TenantName, "{tenant}")
	message = strings.ReplaceAll(message, c.Username, "{username}")
	message = strings.ReplaceAll(message, c.Password, "{password}")
	if c.BaseURL != nil {
		message = strings.ReplaceAll(message, strings.TrimRight(c.BaseURL.Host, "/"), "{host}")
	}

	basicAuthMatcher := regexp.MustCompile(`(Basic\s+)[A-Za-z0-9=]+`)
	message = basicAuthMatcher.ReplaceAllString(message, "$1 {base64 tenant/username:password}")

	oauthMatcher := regexp.MustCompile(`(authorization=)[^\s]+`)
	message = oauthMatcher.ReplaceAllString(message, "$1{OAuth2Token}")

	xsrfTokenMatcher := regexp.MustCompile(`(?i)((X-)?Xsrf-Token:)\s*[^\s]+`)
	message = xsrfTokenMatcher.ReplaceAllString(message, "$1 {xsrfToken}")

	return message
}
