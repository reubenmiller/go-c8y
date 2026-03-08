package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// RequestOptions defines options for making custom HTTP requests.
// This provides backward compatibility with the v1 client API.
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

	// FormData for multipart/form-data uploads
	// map[fieldName]io.Reader
	FormData map[string]io.Reader

	// PrepareRequest allows modifying the resty request before sending
	// This gives access to the full resty API for advanced customization
	PrepareRequest func(*resty.Request)

	OnResponse func(response *resty.Response) error

	DoNotParseResponse bool
}

// RequestResult wraps resty.Response with convenience methods
// for working with Cumulocity IoT API responses.
type RequestResult struct {
	Response   *resty.Response
	Error      error
	cachedBody []byte
}

// IsError returns true if the request failed
func (r *RequestResult) IsError() bool {
	return r.Error != nil || (r.Response != nil && r.Response.IsStatusFailure())
}

// StatusCode returns the HTTP status code
func (r *RequestResult) StatusCode() int {
	if r.Response == nil {
		return 0
	}
	return r.Response.StatusCode()
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
	return r.Response.Header().Get(key)
}

// SendRequest creates and sends a custom HTTP request.
// This method provides backward compatibility with the v1 client
// and is useful for making requests not covered by typed service methods.
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
	req := c.HTTPClient.R().
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

		// Add inline query params from path
		for key, values := range u.Query() {
			for _, value := range values {
				req.SetQueryParam(key, value)
			}
		}
	}

	// Add additional query parameters
	if options.Query != nil {
		switch q := options.Query.(type) {
		case string:
			// Parse string query and add params
			if parsedQuery, err := url.ParseQuery(q); err == nil {
				for key, values := range parsedQuery {
					for _, value := range values {
						req.SetQueryParam(key, value)
					}
				}
			}
		case url.Values:
			req.SetQueryParamsFromValues(q)
		default:
			// Assume struct with url tags
			req.SetQueryParamsFromValues(core.QueryParameters(q))
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
	}

	// Handle multipart form data
	if len(options.FormData) > 0 {
		for fieldName, reader := range options.FormData {
			// Special handling for common Cumulocity binary upload patterns
			if fieldName == "object" {
				// Metadata field - no filename, JSON content type
				req.SetMultipartField(fieldName, "", types.MimeTypeApplicationJSON, reader)
			} else {
				// File field - use field name as filename, octet-stream content type
				req.SetMultipartField(fieldName, fieldName, types.MimeTypeApplicationOctetStream, reader)
			}
		}
	}

	// Override host if provided (use full URL with scheme)
	var fullURL string
	if options.Host != "" {
		// If host override is set, construct full URL
		fullURL = options.Host
		if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
			fullURL = "https://" + fullURL
		}
		fullURL = strings.TrimSuffix(fullURL, "/") + options.Path
	}

	if options.DoNotParseResponse {
		req.SetDoNotParseResponse(true)
	}

	// Prepare request hook
	if options.PrepareRequest != nil {
		options.PrepareRequest(req)
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

	result := &RequestResult{
		Response: resp,
		Error:    err,
	}

	// Call OnResponse callback IMMEDIATELY after receiving response, before any other processing
	// This must happen before we check DoNotParseResponse or errors
	// to allow the user to wrap the raw response body
	if options.OnResponse != nil && resp != nil && err == nil {
		if onResponseErr := options.OnResponse(result.Response); onResponseErr != nil {
			result.Error = onResponseErr
			return result
		}
	}

	if err != nil || options.DoNotParseResponse {
		return result
	}

	// Cache the body immediately so it can be read multiple times
	if resp != nil && resp.Body != nil {
		body, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			result.cachedBody = body
		}
		resp.Body.Close()
	}

	return result
}
