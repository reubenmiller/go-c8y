package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// RequestOptions defines a custom HTTP request for [Client.SendRequest]. It is
// supported v2 API.
//
// Example:
//
//	result := client.SendRequest(ctx, api.RequestOptions{
//	    Method: "GET",
//	    Path:   "/inventory/managedObjects",
//	    Query:  "pageSize=100&type=c8y_Device",
//	})
type RequestOptions struct {
	// Method is the HTTP method (GET, POST, PUT, DELETE, etc.)
	Method string

	// Path is the API path (e.g., "/inventory/managedObjects")
	// Can include query parameters in the path itself
	Path string

	// Query parameters - can be a string, struct with url tags, or url.Values
	Query any

	// Body is the request body - can be any type that resty supports
	Body any

	// Accept header value (defaults to "application/json")
	Accept string

	// ContentType header value
	ContentType string

	// Headers to add to the request
	Headers map[string]string

	// ProcessingMode for C8y CEP processing
	ProcessingMode types.ProcessingMode

	// IgnoreAccept if true, removes the Accept header
	IgnoreAccept bool

	DryRun *bool

	// Host overrides the base URL host for this request
	// Useful for cross-tenant operations or custom aliases
	Host string

	// FormData for multipart/form-data uploads (map[fieldName]io.Reader). The
	// parts are sent in field name order. The value's type sets the part:
	//   - *FormFile: a file part with the given file name and content type
	//   - *FormField: a plain form field
	//   - *os.File: a file part named after the file, with a content type
	//     detected from the file name or the contents
	//   - any other reader: the "object" field is sent as JSON (the binary
	//     metadata of the Cumulocity upload APIs), every other field as a file
	//     part named after the field, as application/octet-stream
	FormData map[string]io.Reader

	// PrepareRequest is called with the final *http.Request just before it is
	// sent, so it can change headers, the URL or the body. It runs once per
	// attempt (again on retries) and also for dry-run requests. Returning an
	// error aborts the request.
	PrepareRequest func(*http.Request) error

	// OnResponse is called with the response before SendRequest returns. With
	// DoNotParseResponse, Body is the unread stream and the hook may replace
	// it (for example to track progress). Otherwise Body is a copy of the
	// cached body, so reading it does not affect RequestResult.Body. An error
	// is returned as RequestResult.Error.
	OnResponse func(response *http.Response) error

	DoNotParseResponse bool
}

// RequestResult is the result of SendRequest. Response and its Request are
// SDK-owned copies: they stay valid after the client sends more requests.
type RequestResult struct {
	// Response is the HTTP response. Its Request field is a copy of the request
	// that was sent, including its body.
	Response *http.Response
	// Error is the error of a request which could not be sent, or a
	// *core.Error for a response with an HTTP error status (400 or above,
	// unless DoNotParseResponse is set).
	Error      error
	cachedBody []byte
}

// IsError returns true if the request failed
func (r *RequestResult) IsError() bool {
	return r.Error != nil || (r.Response != nil && r.Response.StatusCode > 399)
}

// StatusCode returns the HTTP status code
func (r *RequestResult) StatusCode() int {
	if r.Response == nil {
		return 0
	}
	return r.Response.StatusCode
}

// Body returns the response body as bytes.
// The body is cached when the request completes to allow multiple accesses.
func (r *RequestResult) Body() []byte {
	return r.cachedBody
}

// String returns the response body as a string
func (r *RequestResult) String() string {
	return string(r.Body())
}

// JSON parses the response body as JSON using gjson.
// If a path is provided, it returns the value at that path.
// Otherwise, it returns the entire JSON document.
func (r *RequestResult) JSON(path ...string) gjson.Result {
	body := r.Body()
	if len(body) == 0 {
		return gjson.Result{}
	}
	if len(path) > 0 {
		return gjson.GetBytes(body, path[0])
	}
	return gjson.ParseBytes(body)
}

// Unmarshal deserializes the response body into v
func (r *RequestResult) Unmarshal(v any) error {
	if r.Response == nil {
		return fmt.Errorf("no response to unmarshal")
	}
	return json.Unmarshal(r.Body(), v)
}

// Header returns the response header value for the given key
func (r *RequestResult) Header(key string) string {
	if r.Response == nil {
		return ""
	}
	return r.Response.Header.Get(key)
}

// SendRequest creates and sends a custom HTTP request. It is supported v2 API
// and the intended way to call endpoints that have no typed service method; it
// also eases migration from the v1 client. Prefer the typed services where
// they exist, since they return op.Result with metadata and pagination.
//
// Example - Simple GET:
//
//	result := client.SendRequest(ctx, api.RequestOptions{
//	    Method: "GET",
//	    Path:   "/inventory/managedObjects",
//	    Query:  "pageSize=100&type=c8y_Device",
//	})
//	if result.IsError() {
//	    return result.Error
//	}
//	items := result.JSON("managedObjects").Array()
//
// Example - POST with body:
//
//	result := client.SendRequest(ctx, api.RequestOptions{
//	    Method: "POST",
//	    Path:   "/event/events",
//	    Body: map[string]any{
//	        "source": map[string]string{"id": "12345"},
//	        "type":   "testEvent",
//	        "text":   "Test event",
//	    },
//	})
//
// Example - Struct query parameters:
//
//	type ListOptions struct {
//	    PageSize int    `url:"pageSize"`
//	    Type     string `url:"type"`
//	}
//	result := client.SendRequest(ctx, api.RequestOptions{
//	    Method: "GET",
//	    Path:   "/inventory/managedObjects",
//	    Query:  ListOptions{PageSize: 100, Type: "c8y_Device"},
//	})
func (c *Client) SendRequest(ctx context.Context, options RequestOptions) *RequestResult {
	ctxRequest := ctx
	if options.DryRun != nil {
		ctxRequest = WithDryRun(ctxRequest, *options.DryRun)
	}
	if options.PrepareRequest != nil {
		ctxRequest = ctxhelpers.WithRequestHook(ctxRequest, options.PrepareRequest)
	}
	capture := &ctxhelpers.RequestCapture{}
	ctxRequest = ctxhelpers.WithRequestCapture(ctxRequest, capture)
	req := c.httpClient.R().
		SetContext(ctxRequest).
		SetMethod(options.Method)

	// Handle path and query
	if options.Path != "" {
		// Parse path to extract any inline query params
		u, err := url.Parse(options.Path)
		if err != nil {
			return &RequestResult{Error: fmt.Errorf("invalid path: %w", err)}
		}
		req.SetURL(u.Path)

		// Add inline query params from path (every value of a repeated key)
		req.SetQueryParamsFromValues(u.Query())
	}

	// Add additional query parameters
	if options.Query != nil {
		switch q := options.Query.(type) {
		case string:
			// Parse string query and add params
			if parsedQuery, err := url.ParseQuery(q); err == nil {
				req.SetQueryParamsFromValues(parsedQuery)
			}
		case url.Values:
			req.SetQueryParamsFromValues(q)
		default:
			// Assume struct with url tags
			req.SetQueryParamsFromValues(httpcore.QueryParameters(q))
		}
	}

	// Set Accept header
	if !options.IgnoreAccept {
		accept := options.Accept
		if accept == "" {
			accept = types.MimeTypeApplicationJSON
		}
		req.SetHeader("Accept", accept)
	}

	// Apply the context's request options as the typed service methods do:
	// the processing mode and the custom headers and query parameters. The
	// request's own options below win.
	if mode := ctxhelpers.GetProcessingMode(ctxRequest); mode != "" {
		req.SetHeader(types.HeaderProcessingMode, string(mode))
	}
	for key, value := range ctxhelpers.GetExtraHeaders(ctxRequest) {
		req.SetHeader(key, value)
	}
	if options.Accept != "" && !options.IgnoreAccept {
		req.SetHeader("Accept", options.Accept)
	}
	httpcore.SetExtraQueryParams(req, ctxhelpers.GetExtraQueryParams(ctxRequest))

	// Set Content-Type if provided
	if options.ContentType != "" {
		req.SetHeader("Content-Type", options.ContentType)
	}

	// Set custom headers
	for key, value := range options.Headers {
		req.SetHeader(key, value)
	}

	// Set processing mode
	if options.ProcessingMode != "" {
		req.SetHeader(types.HeaderProcessingMode, string(options.ProcessingMode))
	}

	// Set body
	if options.Body != nil {
		req.SetBody(options.Body)
		if strings.EqualFold(options.Method, http.MethodDelete) {
			// The HTTP client drops a DELETE body unless allowed
			req.SetMethodDeleteAllowPayload(true)
		}
	}

	// Handle multipart form data
	if len(options.FormData) > 0 {
		fields, err := multipartFields(options.FormData)
		if err != nil {
			return &RequestResult{Error: fmt.Errorf("invalid form data: %w", err)}
		}
		req.SetMultipartFields(fields...)
	}

	// Override host if provided (use full URL with scheme)
	var fullURL string
	if options.Host != "" {
		// If host override is set, construct full URL
		fullURL = options.Host
		if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
			fullURL = "https://" + fullURL
		}
		// The path's query parameters were already added
		path, _, _ := strings.Cut(options.Path, "?")
		fullURL = strings.TrimSuffix(fullURL, "/") + path
	}

	if options.DoNotParseResponse {
		req.SetResponseDoNotParse(true)
	}

	// Send request (use full URL if host override was set)
	var resp *resty.Response
	var err error
	if fullURL != "" {
		// Clone req and set full URL
		resp, err = req.SetURL(fullURL).Send()
	} else {
		resp, err = req.Send()
	}

	result := &RequestResult{Error: err}
	if resp == nil || resp.RawResponse == nil {
		return result
	}

	httpResp := *resp.RawResponse
	httpResp.Request = capture.Request()
	result.Response = &httpResp

	if err != nil || options.DoNotParseResponse {
		// The caller reads the unread stream from Response.Body.
		httpResp.Body = resp.Body
		if options.OnResponse != nil && err == nil {
			if onResponseErr := options.OnResponse(&httpResp); onResponseErr != nil {
				result.Error = onResponseErr
			}
		}
		return result
	}

	// Cache the body so it can be read multiple times.
	if resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			result.cachedBody = body
		}
		resp.Body.Close()
	}
	httpResp.Body = io.NopCloser(bytes.NewReader(result.cachedBody))

	if httpResp.StatusCode >= 400 {
		apiErr := responseError(resp, &httpResp, result.cachedBody)
		if len(result.cachedBody) == 0 {
			// A JSON error body was parsed by the client: keep it readable
			result.cachedBody = []byte(apiErr.MessageRaw)
			httpResp.Body = io.NopCloser(bytes.NewReader(result.cachedBody))
		}
		result.Error = apiErr
	}

	if options.OnResponse != nil {
		hookResp := httpResp
		hookResp.Body = io.NopCloser(bytes.NewReader(result.cachedBody))
		if onResponseErr := options.OnResponse(&hookResp); onResponseErr != nil {
			result.Error = onResponseErr
		}
	}

	return result
}

// Request sends a custom HTTP request and returns the response as an
// [op.Result] of the JSON body, like the typed service methods do. It is the
// typed escape hatch for endpoints without a service method, and takes the
// same path as them: middleware, authentication, dry run, mock responses,
// context options (tenant, processing mode, request hook, ...) and API error
// handling apply. An HTTP status of 400 or above is returned as the result's
// error (a *core.Error); errors found while building the request are returned
// the same way.
//
// Use [Client.SendRequest] instead when you need the raw response, a streamed
// body (DoNotParseResponse) or the response headers; Request keeps the status,
// duration, request and raw body on the result.
//
// Example:
//
//	result := client.Request(ctx, api.RequestOptions{
//	    Method: "GET",
//	    Path:   "/tenant/system/options/category/key",
//	})
//	if result.IsError() {
//	    return result.Err
//	}
//	value := result.Data.Get("value").String()
func (c *Client) Request(ctx context.Context, options RequestOptions) op.Result[jsondoc.JSONDoc] {
	raw := c.SendRequest(ctx, options)

	var httpReq *http.Request
	status := raw.StatusCode()
	if raw.Response != nil {
		httpReq = raw.Response.Request
	}

	var result op.Result[jsondoc.JSONDoc]
	switch {
	case raw.Error != nil:
		result = op.Failed[jsondoc.JSONDoc](raw.Error, false)
	case status == http.StatusCreated:
		result = op.Created(jsondoc.New(raw.Body()))
	case status == http.StatusOK && (options.Method == http.MethodPut || options.Method == http.MethodPatch):
		result = op.Updated(jsondoc.New(raw.Body()))
	default:
		result = op.OK(jsondoc.New(raw.Body()))
	}
	if raw.Response != nil {
		result = op.WithHTTPStatus(result, status)
		result = op.WithRequestID(result, raw.Header("X-Request-ID"))
	}
	result = op.WithRequest(result, httpReq)
	if raw.Error == nil {
		result = op.WithResponse(result, raw.Body())
	}
	return result
}
