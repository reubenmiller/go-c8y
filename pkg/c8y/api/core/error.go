package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"time"

	"resty.dev/v3"
)

const (
	ErrorUnsupported = iota
	// ErrorFromString is the Code identifying Errors created by string types
	ErrorFromString
	// ErrorFromError is the Code identifying Errors created by error types
	ErrorFromError
	// ErrorFromStringer is the Code identifying Errors created by fmt.Stringer types
	ErrorFromStringer
)

// Error wraps the Cumulocity Server error with the relevant http.Response
type Error struct {
	Response   *http.Response
	Code       int
	Type       string
	Message    string
	MessageRaw string

	ReceivedAt time.Time
	Duration   time.Duration
}

// APIError is the error-set returned by Cumulocity API when presented with an invalid request
type APIError struct {
	ErrorType string `json:"error,omitempty"`   // Error type formatted as "<<resource type>>/<<error name>>"". For example, an object not found in the inventory is reported as "inventory/notFound".
	Message   string `json:"message,omitempty"` // error message
	Info      string `json:"info,omitempty"`    // URL to an error description on the Internet.
}

func (r APIError) Error() string {
	return r.Message
}

func coupleAPIErrors(r *resty.Response, err error) (*resty.Response, error) {
	if err != nil {
		// an error was raised in go code, no need to check the resty Response
		return nil, NewError(err)
	}

	if r.IsStatusSuccess() {
		return r, nil
	}

	// TODO: Check why this logic exists, as it seems wrong to me
	// if r.IsRead && r.Error() == nil {
	// 	// no error in the resty Response
	// 	return r, nil
	// }

	// Check that response is of the correct content-type before unmarshalling
	// expectedContentType := r.Request.Header.Get("Accept")
	responseContentType := r.Header().Get("Content-Type")

	// If the upstream Cumulocity API server being fronted fails to respond to the request,
	// the http server will respond with a default "Bad Gateway" page with Content-Type
	// "text/html".
	if r.StatusCode() == http.StatusBadGateway && responseContentType == "text/html" {
		return nil, Error{Code: http.StatusBadGateway, Message: http.StatusText(http.StatusBadGateway)}
	}

	if !r.IsRead {
		// Manually parse the response to analyze the error message
		defer r.Body.Close()
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, NewError(err)
		}

		var apiError APIError
		if err := json.Unmarshal(buf, &apiError); err != nil {
			return nil, NewError(err)
		}

		return nil, &Error{
			Code: r.StatusCode(),

			Type:       apiError.ErrorType,
			Message:    apiError.Error(),
			MessageRaw: string(buf),
			Response:   r.RawResponse,
			Duration:   r.Duration(),
			ReceivedAt: r.ReceivedAt(),
		}
	}

	return nil, NewError(r)
}

// NewError creates a Cumulocity Server.Error with a Code identifying the source err type,
// - ErrorFromString   (1) from a string
// - ErrorFromError    (2) for an error
// - ErrorFromStringer (3) for a Stringer
// - HTTP Status Codes (100-600) for a resty.Response object
func NewError(err any) *Error {
	if err == nil {
		return nil
	}

	switch e := err.(type) {
	case *Error:
		return e
	case *resty.Response:
		apiError, ok := e.ResultError().(*APIError)

		if !ok {
			// If Build error manually if the APIError wasn't serialized
			wrappedErr := &Error{
				Code:       e.StatusCode(),
				Type:       "NoContent",
				Message:    e.String(),
				MessageRaw: e.String(),
				Response:   e.RawResponse,
				Duration:   e.Duration(),
				ReceivedAt: e.ReceivedAt(),
			}
			if e.CascadeError != nil {
				wrappedErr.Message = e.CascadeError.Error()
			}
			return wrappedErr
		}

		return &Error{
			Code: e.StatusCode(),

			Type:       apiError.ErrorType,
			Message:    apiError.Error(),
			MessageRaw: e.String(),
			Response:   e.RawResponse,
			Duration:   e.Duration(),
			ReceivedAt: e.ReceivedAt(),
		}
	case error:
		return &Error{Code: ErrorFromError, Message: e.Error()}
	case string:
		return &Error{Code: ErrorFromString, Message: e}
	case fmt.Stringer:
		return &Error{Code: ErrorFromStringer, Message: e.String()}
	default:
		return &Error{Code: ErrorUnsupported, Message: fmt.Sprintf("Unsupported type to Cumulocity Server.NewError: %s", reflect.TypeOf(e))}
	}
}

func (err Error) Error() string {
	if err.Response == nil || err.Response.Request == nil {
		return fmt.Sprintf("[%03d] %v : %v", err.StatusCode(), err.Type, err.Message)
	}
	return fmt.Sprintf("[%03d] %v : %v", err.StatusCode(), err.Type, err.Message)
}

func (err Error) StatusCode() int {
	return err.Code
}

func (err Error) Is(target error) bool {
	if x, ok := target.(interface{ StatusCode() int }); ok || errors.As(target, &x) {
		return err.StatusCode() == x.StatusCode()
	}

	return false
}

// IsNotFound indicates if err indicates a 404 Not Found error from the Cumulocity API.
func IsNotFound(err error) bool {
	return ErrHasStatus(err, http.StatusNotFound)
}

// ErrHasStatus checks if err is an error from the Cumulocity API, and whether it contains the given HTTP status code.
// More than one status code may be given.
// If len(code) == 0, err is nil or is not a [Error], ErrHasStatus will return false.
func ErrHasStatus(err error, code ...int) bool {
	if err == nil {
		return false
	}

	// Short-circuit if the caller did not provide any status codes.
	if len(code) == 0 {
		return false
	}

	var e *Error
	if !errors.As(err, &e) {
		return false
	}

	ec := e.StatusCode()
	for _, c := range code {
		if ec == c {
			return true
		}
	}

	return false
}

func ErrTokenRevoked(err any) bool {
	// Token '{uuid}' not present for user
	// Token is terminated
	if apiError, ok := err.(*APIError); ok {
		notPresent := regexp.MustCompile("(?i)^Token '[0-9a-z-]+' not present for user")
		if notPresent.MatchString(apiError.Message) {
			return true
		}

		tokenTerminated := regexp.MustCompile("(?i)^Token is terminated")
		if tokenTerminated.MatchString(apiError.Message) {
			return true
		}
	}
	return false
}

// getStatusCodeFromError extracts the HTTP status code from an error if it's an *Error type
func getStatusCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var e *Error
	if errors.As(err, &e) {
		return e.StatusCode()
	}
	return 0
}
