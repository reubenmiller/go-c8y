package c8y

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/google/go-querystring/query"
	"github.com/tidwall/gjson"
)

// ContextAuthTokenKey todo
type ContextAuthTokenKey string

// GetContextAuthTokenKey authentication key used to override the given Basic Authentication token
func GetContextAuthTokenKey() ContextAuthTokenKey {
	return ContextAuthTokenKey("authToken")
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

	UseTenantInUsername bool

	// Microservice bootstrap and service users
	BootstrapUser ServiceUser
	ServiceUsers  []ServiceUser

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the Cumulocity API.
	Context      *ContextService
	Alarm        *AlarmService
	Measurement  *MeasurementService
	Operation    *OperationService
	Tenant       *TenantService
	Event        *EventService
	Inventory    *InventoryService
	Application  *ApplicationService
	Identity     *IdentityService
	Microservice *MicroserviceService
}

const (
	defaultUserAgent = "go-client"
)

// DecodeJSONBytes decodes json preserving number formatting (especially large integers and scientific notation floats)
func DecodeJSONBytes(v []byte, dst interface{}) error {
	return DecodeJSONReader(bytes.NewReader(v), dst)
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
		log.Panic("No service users found")
	}
	for _, user := range c.ServiceUsers {
		if tenant == user.Tenant || tenant == "" {
			return NewRealtimeClient(c.BaseURL.String(), nil, user.Tenant, user.Username, user.Password)
		}
	}
	return nil
}

// NewClientUsingBootstrapUserFromEnvironment returns a Cumulocity client using the the bootstrap credentials set in the environment variables
func NewClientUsingBootstrapUserFromEnvironment(httpClient *http.Client, baseURL string) *Client {
	tenant, username, password := GetBootstrapUserFromEnvironment()

	client := NewClient(httpClient, baseURL, tenant, username, password, true)
	client.Microservice.SetServiceUsers()

	// TODO: Setup a realtime client
	// if !skipRealtimeClient {
	// 	client.clientMu.Lock()
	// 	client.Realtime =
	// 	client.clientMu.Unlock()
	// }

	return client
}

// NewClient returns a new Cumulocity API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(httpClient *http.Client, baseURL string, tenant string, username string, password string, skipRealtimeClient bool) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
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
		log.Printf("Creating realtime client %s\n", fmtURL)
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
	c.Measurement = (*MeasurementService)(&c.common)
	c.Operation = (*OperationService)(&c.common)
	c.Tenant = (*TenantService)(&c.common)
	c.Event = (*EventService)(&c.common)
	c.Inventory = (*InventoryService)(&c.common)
	c.Application = (*ApplicationService)(&c.common)
	c.Identity = (*IdentityService)(&c.common)
	c.Microservice = (*MicroserviceService)(&c.common)
	c.Context = (*ContextService)(&c.common)
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
	rawQuery = rawQuery[1:len(rawQuery)]
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
	Method       string
	Host         string
	Path         string
	Query        interface{} // Use string if you want
	Body         interface{}
	ResponseData interface{}
}

// SendRequest creates and sends a request
func (c *Client) SendRequest(ctx context.Context, options RequestOptions) (*Response, error) {

	queryParams := ""

	if options.Query != nil {
		if v, ok := options.Query.(string); ok {
			queryParams = v
		} else {
			if v, err := addOptions("", options.Query); err == nil {
				queryParams = v
			} else {
				log.Printf("ERROR: Could not convert query parameter interface{} to a string. %s", err)
				return nil, err
			}
		}
	}

	req, err := c.NewRequest(options.Method, options.Path, queryParams, options.Body)

	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	if options.Host != "" {
		log.Printf("Using alternative host %s", options.Host)
		req.Host = options.Host
	}

	if err != nil {
		return nil, err
	}

	resp, err := c.Do(ctx, req, options.ResponseData)
	if err != nil {
		return resp, err
	}
	return resp, nil
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
		req.Header.Set("Content-Type", "application/json")
	}

	c.SetAuthorization(req)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-APPLICATION", "go-client")
	return req, nil
}

// SetAuthorization sets the configured authorization to the given request. By default it will set the Basic Authorization header
func (c *Client) SetAuthorization(req *http.Request) {
	var headerUsername string
	if c.UseTenantInUsername {
		headerUsername = fmt.Sprintf("%s/%s", c.TenantName, c.Username)
	} else {
		headerUsername = c.Username
	}
	req.SetBasicAuth(headerUsername, c.Password)
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

type apiCategory string

// category returns the rate limit category of the endpoint, determined by Request.URL.Path.
func category(path string) apiCategory {
	switch {
	default:
		return "unknown"
	case strings.HasPrefix(path, "/measurement/"):
		return "measurement"
	}
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
		log.Printf("Overriding basic auth provided in the context\n")
		req.Header.Set("Authorization", authToken.(string))
	}

	log.Println("Sending request: ", req.URL.Path)

	resp, err := c.client.Do(req)
	if err != nil {
		// If we got an error, and the context has been canceled,
		// the context's error is probably more useful.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// If the error type is *url.Error, sanitize its URL before returning.
		if e, ok := err.(*url.Error); ok {
			if url, err := url.Parse(e.URL); err == nil {
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
		log.Printf("Invalid response received from server. %s", err)
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			err = DecodeJSONReader(resp.Body, v)

			if err == io.EOF {
				log.Printf("Error decoding body. %s", err)
				err = nil // ignore EOF errors caused by empty response body
			}
		}
	}

	log.Println(fmt.Sprintf("Status code: %v", response.StatusCode))

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
	Response *http.Response // HTTP response that caused this error
	Message  string         `json:"message"` // error message
	Errors   []Error        `json:"errors"`  // more detail on individual errors
	// Block is only populated on certain types of errors such as code 451.
	// See https://developer.github.com/changes/2016-03-17-the-451-status-code-is-now-supported/
	// for more information.
	Block *struct {
		Reason    string     `json:"reason,omitempty"`
		CreatedAt *Timestamp `json:"created_at,omitempty"`
	} `json:"block,omitempty"`
	// Most errors will also include a documentation_url field pointing
	// to some content that might help you resolve the error, see
	// https://developer.github.com/v3/#client-errors
	DocumentationURL string `json:"documentation_url,omitempty"`
}

func (r *ErrorResponse) Error() string {
	return fmt.Sprintf("%v %v: %d %v %+v",
		r.Response.Request.Method, sanitizeURL(r.Response.Request.URL),
		r.Response.StatusCode, r.Message, r.Errors)
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
