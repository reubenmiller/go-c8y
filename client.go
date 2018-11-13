package go-c8y

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/google/go-querystring/query"
)

type service struct {
	client *Client
}

// A Client manages communication with the GitHub API.
type Client struct {
	clientMu sync.Mutex   // clientMu protects the client during calls that modify the CheckRedirect func.
	client   *http.Client // HTTP client used to communicate with the API.

	// Base URL for API requests. Defaults to the public GitHub API, but can be
	// set to a domain endpoint to use with GitHub Enterprise. BaseURL should
	// always be specified with a trailing slash.
	BaseURL *url.URL

	// User agent used when communicating with the GitHub API.
	UserAgent string

	rateMu sync.Mutex
	// rateLimits [categories]Rate // Rate limits for the client as determined by the most recent API calls.

	// Username for Cumulocity Authentication
	Username string

	// Cumulocity Tenant
	TenantName string

	// Password for Cumulocity Authentication
	Password string

	verboseMessage *color.Color
	warningMessage *color.Color

	common service // Reuse a single struct instead of allocating one for each service on the heap.

	// Services used for talking to different parts of the GitHub API.
	//Activity       *ActivityService
	Measurement *MeasurementService
	Operation   *OperationService
	Tenant      *TenantService
	Event       *EventService
	Inventory   *InventoryService
	Application *ApplicationService
	Identity    *IdentityService
}

// Client is an example of the c8y client
/* type Client struct {
	BaseURL   *url.URL
	UserAgent string

	httpClient *http.Client

	common service

	Measurement *MeasurementService
} */

const (
	defaultUserAgent = "go-client"
)

// NewClient returns a new GitHub API client. If a nil httpClient is
// provided, http.DefaultClient will be used. To use API methods which require
// authentication, provide an http.Client that will perform the authentication
// for you (such as that provided by the golang.org/x/oauth2 library).
func NewClient(httpClient *http.Client, baseURL string, username string, password string) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	targetBaseURL, _ := url.Parse(baseURL)

	usernameParts := strings.Split(username, "/")

	var tenantName string
	if len(usernameParts) == 2 {
		tenantName = usernameParts[0]
	}

	userAgent := defaultUserAgent

	c := &Client{
		client:         httpClient,
		BaseURL:        targetBaseURL,
		UserAgent:      userAgent,
		Username:       username,
		Password:       password,
		TenantName:     tenantName,
		verboseMessage: color.New(color.FgMagenta),
		warningMessage: color.New(color.FgYellow),
	}
	c.common.client = c
	c.Measurement = (*MeasurementService)(&c.common)
	c.Operation = (*OperationService)(&c.common)
	c.Tenant = (*TenantService)(&c.common)
	c.Event = (*EventService)(&c.common)
	c.Inventory = (*InventoryService)(&c.common)
	c.Application = (*ApplicationService)(&c.common)
	c.Identity = (*IdentityService)(&c.common)
	return c
}

func meTest() {
	println("test api")
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

	// c := color.New(color.FgMagenta)
	// c.Println("rawQuery: ", u.RawQuery)

	rawQuery := u.String()
	rawQuery = rawQuery[1:len(rawQuery)]
	// c.Println("query: ", rawQuery)
	return rawQuery, nil
}

// Noop todo
func (c *Client) Noop() {

}

// NewRequest does something
func (c *Client) NewRequest(method, path string, query string, body interface{}) (*http.Request, error) {
	// c.verboseMessage.Println("newRequest", path)

	if !strings.HasSuffix(c.BaseURL.Path, "/") {
		return nil, fmt.Errorf("BaseURL must have a trailing slash, but %q does not", c.BaseURL)
	}

	// c.verboseMessage.Println("Before url: ", path)
	// rel := &url.URL{Opaque: path}
	rel := &url.URL{Path: path}
	if query != "" {
		rel.RawQuery = query
	}
	// c.verboseMessage.Println("Before resolve: ", path, rel.String())
	u := c.BaseURL.ResolveReference(rel)

	// c.verboseMessage.Println("After resolve: ", u.String())

	var buf io.ReadWriter
	if body != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(body)
		if err != nil {
			return nil, err
		}
	}
	req, err := http.NewRequest(method, u.String(), buf)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	req.SetBasicAuth(c.Username, c.Password)

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("X-NX-APPLICATION", "nx-nif-quality-gate")
	return req, nil
}

type measurement struct {
	ID string `json:"id"`
}

// Rate represents the rate limit for the current client.
type Rate struct {
	// The number of requests per hour the client is currently limited to.
	Measurements []measurement `json:"measurements"`

	// The number of remaining requests the client can make this hour.
	Remaining int `json:"remaining"`

	// The time at which the current rate limit will reset.
	// Reset Timestamp `json:"reset"`
}

// Response is a GitHub API response. This wraps the standard http.Response
// returned from GitHub and provides convenient access to things like
// pagination links.
type Response struct {
	*http.Response

	// JSONData raw json response
	JSONData *string
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
// first decode it. If rate limit is exceeded and reset time is in the future,
// Do returns *RateLimitError immediately without making a network API call.
//
// The provided ctx must be non-nil. If it is canceled or times out,
// ctx.Err() will be returned.
func (c *Client) Do(ctx context.Context, req *http.Request, v interface{}) (*Response, error) {
	req = withContext(ctx, req)

	c.verboseMessage.Println("Sending request: ", req.URL.Path)

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

	defer func() {
		// Drain up to 512 bytes and close the body to let the Transport reuse the connection
		io.CopyN(ioutil.Discard, resp.Body, 512)
		resp.Body.Close()
	}()

	response := newResponse(resp)

	c.rateMu.Lock()
	// c.rateLimits[rateLimitCategory] = response.Rate
	c.rateMu.Unlock()

	err = CheckResponse(resp)
	if err != nil {
		// even though there was an error, we still return the response
		// in case the caller wants to inspect it further
		println("Invalid response received from server")
		return response, err
	}

	if v != nil {
		if w, ok := v.(io.Writer); ok {
			io.Copy(w, resp.Body)
		} else {
			err = json.NewDecoder(resp.Body).Decode(v)
			if err == io.EOF {
				println("error decoding body")
				err = nil // ignore EOF errors caused by empty response body
			}
		}
	}

	c.verboseMessage.Println(fmt.Sprintf("Status code: %v", response.StatusCode))

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
GitHub API docs: https://developer.github.com/v3/#client-errors
*/
type Error struct {
	Resource string `json:"resource"` // resource on which the error occurred
	Field    string `json:"field"`    // field on which the error occurred
	Code     string `json:"code"`     // validation error code
	Message  string `json:"message"`  // Message describing the error. Errors with Code == "custom" will always have this set.
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v error caused by %v field on %v resource",
		e.Code, e.Field, e.Resource)
}

/*
An ErrorResponse reports one or more errors caused by an API request.
GitHub API docs: https://developer.github.com/v3/#client-errors
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

// AcceptedError occurs when GitHub returns 202 Accepted response with an
// empty body, which means a job was scheduled on the GitHub side to process
// the information needed and cache it.
// Technically, 202 Accepted is not a real error, it's just used to
// indicate that results are not ready yet, but should be available soon.
// The request can be repeated after some time.
type AcceptedError struct{}

func (*AcceptedError) Error() string {
	return "job scheduled on GitHub side; try again later"
}

// CheckResponse checks the API response for errors, and returns them if
// present. A response is considered an error if it has a status code outside
// the 200 range or equal to 202 Accepted.
// API error responses are expected to have either no response
// body, or a JSON response body that maps to ErrorResponse. Any other
// response body will be silently ignored.
//
// The error type will be *RateLimitError for rate limit exceeded errors,
// *AcceptedError for 202 Accepted status codes,
// and *TwoFactorAuthError for two-factor authentication errors.
func CheckResponse(r *http.Response) error {
	if r.StatusCode == http.StatusAccepted {
		return &AcceptedError{}
	}
	if c := r.StatusCode; 200 <= c && c <= 299 {
		return nil
	}
	errorResponse := &ErrorResponse{Response: r}
	data, err := ioutil.ReadAll(r.Body)
	// text := string(data)
	if err == nil && data != nil {
		json.Unmarshal(data, errorResponse)
	}
	switch {
	/* case r.StatusCode == http.StatusUnauthorized && strings.HasPrefix(r.Header.Get(headerOTP), "required"):
	return (*TwoFactorAuthError)(errorResponse) */
	/* case r.StatusCode == http.StatusForbidden && r.Header.Get(headerRateRemaining) == "0" && strings.HasPrefix(errorResponse.Message, "API rate limit exceeded for "):
	return &RateLimitError{
		Rate:     parseRate(r),
		Response: errorResponse.Response,
		Message:  errorResponse.Message,
	} */
	/* case r.StatusCode == http.StatusForbidden && strings.HasSuffix(errorResponse.DocumentationURL, "/v3/#abuse-rate-limits"):
	abuseRateLimitError := &AbuseRateLimitError{
		Response: errorResponse.Response,
		Message:  errorResponse.Message,
	}
	if v := r.Header["Retry-After"]; len(v) > 0 {
		// According to GitHub support, the "Retry-After" header value will be
		// an integer which represents the number of seconds that one should
		// wait before resuming making requests.
		retryAfterSeconds, _ := strconv.ParseInt(v[0], 10, 64) // Error handling is noop.
		retryAfter := time.Duration(retryAfterSeconds) * time.Second
		abuseRateLimitError.RetryAfter = &retryAfter
	}
	return abuseRateLimitError */
	default:
		return errorResponse
	}
}
