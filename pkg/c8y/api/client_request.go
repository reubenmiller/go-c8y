package api

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// RequestOptions defines a custom HTTP request for [Client.Request].
//
// Example:
//
//	result := client.Request(ctx, api.RequestOptions{
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

	// OnResponse is called with the response before Request returns. Its Body
	// is a copy of the buffered body, so reading it does not affect the
	// result. An error is returned as the result's error.
	OnResponse func(response *http.Response) error
}

// rawResponse is the outcome of a request made by Client.Request. Response and
// its Request are SDK-owned copies: they stay valid after the client sends more
// requests.
type rawResponse struct {
	// response is the HTTP response, or nil when none was received (a failed
	// or dry-run request).
	response *http.Response
	// request is the request that was sent, or would be sent in a dry run.
	request *http.Request
	// err is the error of a request which could not be sent, or a *core.Error
	// for a response with an HTTP error status (400 or above).
	err  error
	body []byte
}

func (r *rawResponse) statusCode() int {
	if r.response == nil {
		return 0
	}
	return r.response.StatusCode
}

// send builds and sends the request of options and returns the buffered response.
func (c *Client) send(ctx context.Context, options RequestOptions) *rawResponse {
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
			return &rawResponse{err: fmt.Errorf("invalid path: %w", err)}
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
			return &rawResponse{err: fmt.Errorf("invalid form data: %w", err)}
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

	// Send request (use full URL if host override was set)
	var resp *resty.Response
	var err error
	if fullURL != "" {
		// Clone req and set full URL
		resp, err = req.SetURL(fullURL).Send()
	} else {
		resp, err = req.Send()
	}

	result := &rawResponse{err: err, request: capture.Request()}
	if resp == nil || resp.RawResponse == nil {
		return result
	}

	httpResp := *resp.RawResponse
	httpResp.Request = result.request
	result.response = &httpResp

	if err != nil {
		httpResp.Body = resp.Body
		return result
	}

	// Buffer the body so it can be read more than once.
	if resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			result.body = body
		}
		resp.Body.Close()
	}
	httpResp.Body = io.NopCloser(bytes.NewReader(result.body))

	if httpResp.StatusCode >= 400 {
		apiErr := responseError(resp, &httpResp, result.body)
		if len(result.body) == 0 {
			// A JSON error body was parsed by the client: keep it readable
			result.body = []byte(apiErr.MessageRaw)
			httpResp.Body = io.NopCloser(bytes.NewReader(result.body))
		}
		result.err = apiErr
	}

	if options.OnResponse != nil {
		hookResp := httpResp
		hookResp.Body = io.NopCloser(bytes.NewReader(result.body))
		if onResponseErr := options.OnResponse(&hookResp); onResponseErr != nil {
			result.err = onResponseErr
		}
	}

	return result
}

// Request sends a custom HTTP request and returns the response as an
// [op.Result] of the JSON body, like the typed service methods do. It is the
// intended way to call endpoints that have no typed service method. It takes
// the same path as the typed services: middleware, authentication, dry run,
// mock responses, context options (tenant, processing mode, request hook, ...)
// and API error handling apply. An HTTP status of 400 or above is returned as
// the result's error (a *core.Error); errors found while building the request
// are returned the same way.
//
// The result keeps what a caller of a raw request needs: HTTPStatus, the
// response Header, the raw body in Response (also when it is not JSON, and for an HTTP error), the
// Request that was sent (or would be, in a dry run).
//
// Example:
//
//	result := client.Request(ctx, api.RequestOptions{
//	    Method: "GET",
//	    Path:   "/inventory/managedObjects",
//	    Query:  "pageSize=1",
//	})
//	if result.IsError() {
//	    return result.Err
//	}
//	name := result.Data.Get("managedObjects.0.name").String()
//
// Example, a POST with a body:
//
//	result := client.Request(ctx, api.RequestOptions{
//	    Method: "POST",
//	    Path:   "/event/events",
//	    Body: map[string]any{
//	        "source": map[string]string{"id": "12345"},
//	        "type":   "testEvent",
//	        "text":   "Test event",
//	    },
//	})
//
// Example, struct query parameters:
//
//	type ListOptions struct {
//	    PageSize int    `url:"pageSize"`
//	    Type     string `url:"type"`
//	}
//	result := client.Request(ctx, api.RequestOptions{
//	    Method: "GET",
//	    Path:   "/inventory/managedObjects",
//	    Query:  ListOptions{PageSize: 100, Type: "c8y_Device"},
//	})
func (c *Client) Request(ctx context.Context, options RequestOptions) op.Result[jsondoc.JSONDoc] {
	raw := c.send(ctx, options)
	status := raw.statusCode()

	var result op.Result[jsondoc.JSONDoc]
	switch {
	case raw.err != nil:
		result = op.Failed[jsondoc.JSONDoc](raw.err, false)
	case status == http.StatusCreated:
		result = op.Created(jsondoc.New(raw.body))
	case status == http.StatusOK && (options.Method == http.MethodPut || options.Method == http.MethodPatch):
		result = op.Updated(jsondoc.New(raw.body))
	default:
		result = op.OK(jsondoc.New(raw.body))
	}
	if raw.response != nil {
		result = op.WithHTTPStatus(result, status)
		result = op.WithRequestID(result, raw.response.Header.Get("X-Request-ID"))
		result.Header = raw.response.Header
	}
	result = op.WithRequest(result, raw.request)
	result = op.WithResponse(result, raw.body)
	return result
}
