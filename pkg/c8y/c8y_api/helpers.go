package c8y_api

import (
	"context"
	"encoding/json"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/tidwall/gjson"
	"resty.dev/v3"
)

// ForEach iterates over each result and will fetch additional results from the server if required
// The user can provide their own data type which is to be used on each result
func ForEach[A any](ctx context.Context, r *core.TryRequest, pagerOpts pagination.PagerOptions, out chan<- A) error {
	return pagination.ForEach(ctx, r, pagerOpts, out)
}

// ForEach iterates over each result in raw json format. It will fetch additional results from the server if required
func ForEachJSON[A any](ctx context.Context, r *core.TryRequest, pagerOpts pagination.PagerOptions, out chan<- gjson.Result) error {
	return pagination.ForEachJSON(ctx, r, pagerOpts, out)
}

// Execute a request and return the typed response
func Execute[T any](ctx context.Context, req *core.TryRequest) (*T, *resty.Response, error) {
	return core.Execute[T](ctx, req)
}

// Remove Accept Header
func NoAcceptHeader(r *resty.Request) *resty.Request {
	r.Header.Del("Accept")
	return r
}

func ExecuteNoResult(ctx context.Context, req *core.TryRequest) error {
	return core.ExecuteNoResult(ctx, req)
}

func ExecuteResponseOnly(ctx context.Context, req *core.TryRequest) (*resty.Response, error) {
	return core.ExecuteResponseOnly(ctx, req)
}

func UnmarshalJSON(req *core.Response, data any) error {
	dec := json.NewDecoder(req.Response.Body)
	dec.UseNumber()
	return dec.Decode(&data)
}

func ErrHasStatus(err error, code ...int) bool {
	return core.ErrHasStatus(err, code...)
}

type Error = core.Error

// The request was unacceptable, often due to missing a required parameter
var ErrBadRequest = Error{Code: 400}

// The request was unacceptable, often due to missing a required parameter
var ErrUnauthorized = Error{Code: 401}

// Authentication has failed, or credentials were required but not provided.
var ErrForbidden = Error{Code: 403}

// TODO: Resolve with ErrNotFound
// The requested resource doesn't exist
var ErrAPINotFound = Error{Code: 404}

// The employed HTTP method cannot be used on this resource (for example, using PUT on a read-only resource)
var ErrMethodNotAllowed = Error{Code: 405}

// The server could not produce a response matching the list of accepted types defined in the request
var ErrNotAcceptable = Error{Code: 406}

// The data is correct but it breaks some constraints (for example, application version limit is exceeded)
var ErrConflict = Error{Code: 409}

// Invalid data was sent on the request and/or a query could not be understood.
var ErrInvalidData = Error{Code: 422}

// The requested resource cannot be updated or mandatory fields are missing on the executed operation.
var ErrUnprocessableEntity = Error{Code: 422}

// Something went wrong on Cumulocity's end.
var ErrServer500 = Error{Code: 500}

// Something went wrong on Cumulocity's end.
var ErrServer501 = Error{Code: 501}

// Something went wrong on Cumulocity's end.
var ErrServer502 = Error{Code: 502}

// Something went wrong on Cumulocity's end.
var ErrServer503 = Error{Code: 503}
