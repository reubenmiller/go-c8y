package api

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

func ErrHasStatus(err error, code ...int) bool {
	return core.ErrHasStatus(err, code...)
}

// IsNotFound reports whether err is a 404 Not Found error, including the "no
// result" error of the op package.
func IsNotFound(err error) bool {
	return core.IsNotFound(err)
}

// IsUnauthorized reports whether err is an HTTP 401 error.
func IsUnauthorized(err error) bool {
	return core.ErrHasStatus(err, http.StatusUnauthorized)
}

// IsForbidden reports whether err is a 403 Forbidden error.
func IsForbidden(err error) bool {
	return core.ErrHasStatus(err, http.StatusForbidden)
}

// IsConflict reports whether err is a 409 Conflict error.
func IsConflict(err error) bool {
	return core.ErrHasStatus(err, http.StatusConflict)
}

// StatusCode returns the HTTP status code carried by err (or an error it
// wraps). The second result is false when err has no HTTP status, for example
// a transport failure or a validation error.
func StatusCode(err error) (int, bool) {
	var e *core.Error
	if errors.As(err, &e) && e.Code >= 100 && e.Code < 600 {
		return e.Code, true
	}
	var v core.Error
	if errors.As(err, &v) && v.Code >= 100 && v.Code < 600 {
		return v.Code, true
	}
	return 0, false
}

type Error = core.Error

var (
	// The request was unacceptable, often due to missing a required parameter
	ErrBadRequest = Error{Code: 400}

	// The request was unacceptable, often due to missing a required parameter
	ErrUnauthorized = Error{Code: 401}

	// Authentication has failed, or credentials were required but not provided.
	ErrForbidden = Error{Code: 403}

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
