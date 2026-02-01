package c8y_api

import (
	"context"
	"encoding/json"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"resty.dev/v3"
)

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

var (
	// The request was unacceptable, often due to missing a required parameter
	ErrBadRequest = Error{Code: 400}

	// The request was unacceptable, often due to missing a required parameter
	ErrUnauthorized = Error{Code: 401}

	// Authentication has failed, or credentials were required but not provided.
	ErrForbidden = Error{Code: 403}

	// The requested resource doesn't exist
	// TODO: Resolve with ErrNotFound
	ErrAPINotFound = Error{Code: 404}

	// The employed HTTP method cannot be used on this resource (for example, using PUT on a read-only resource)
	ErrMethodNotAllowed = Error{Code: 405}

	// The server could not produce a response matching the list of accepted types defined in the request
	ErrNotAcceptable = Error{Code: 406}

	// The data is correct but it breaks some constraints (for example, application version limit is exceeded)
	ErrConflict = Error{Code: 409}

	// Invalid data was sent on the request and/or a query could not be understood.
	ErrInvalidData = Error{Code: 422}

	// The requested resource cannot be updated or mandatory fields are missing on the executed operation.
	ErrUnprocessableEntity = Error{Code: 422}

	// Something went wrong on Cumulocity's end.
	ErrServer500 = Error{Code: 500}

	// Something went wrong on Cumulocity's end.
	ErrServer501 = Error{Code: 501}

	// Something went wrong on Cumulocity's end.
	ErrServer502 = Error{Code: 502}

	// Something went wrong on Cumulocity's end.
	ErrServer503 = Error{Code: 503}
)
