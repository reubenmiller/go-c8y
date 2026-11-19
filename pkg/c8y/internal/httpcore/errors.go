package httpcore

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

func coupleAPIErrors(r *resty.Response, err error) (*resty.Response, error) {
	if err != nil {
		// an error was raised in go code, no need to check the resty Response
		return nil, core.NewError(err)
	}

	if r.IsStatusSuccess() {
		return r, nil
	}

	// Check that response is of the correct content-type before unmarshalling
	// expectedContentType := r.Request.Header.Get("Accept")
	responseContentType := r.Header().Get("Content-Type")

	// If the upstream Cumulocity API server being fronted fails to respond to the request,
	// the http server will respond with a default "Bad Gateway" page with Content-Type
	// "text/html".
	if r.StatusCode() == http.StatusBadGateway && responseContentType == "text/html" {
		return nil, core.Error{Code: http.StatusBadGateway, Message: http.StatusText(http.StatusBadGateway)}
	}

	if !r.IsRead {
		// Manually parse the response to analyse the error message
		defer r.Body.Close()
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, core.NewError(err)
		}

		var apiError core.APIError
		if err := json.Unmarshal(buf, &apiError); err != nil {
			return nil, core.NewError(err)
		}

		httpErr := &core.Error{
			Code: r.StatusCode(),

			Type:       apiError.ErrorType,
			Message:    apiError.Error(),
			MessageRaw: string(buf),
			Response:   r.RawResponse,
			Duration:   r.Duration(),
			ReceivedAt: r.ReceivedAt(),
		}
		if be := core.NewBatchError(httpErr, apiError.BulkRequest); be != nil {
			return nil, be
		}
		return nil, httpErr
	}

	httpErr := core.NewError(r)
	if apiError, ok := r.ResultError().(*core.APIError); ok && apiError != nil {
		if be := core.NewBatchError(httpErr, apiError.BulkRequest); be != nil {
			return nil, be
		}
	}
	return nil, httpErr
}

// getStatusCodeFromError extracts the HTTP status code from an error if it's an *Error type
func getStatusCodeFromError(err error) int {
	if err == nil {
		return 0
	}
	var e *core.Error
	if errors.As(err, &e) {
		return e.StatusCode()
	}
	return 0
}

// failed builds a failed result. Batch errors (see core.BatchError) are never
// retryable, because the platform says to never resend a failed bulk request,
// and get op.StatusPartial when part of the request was written.
func failed[T any](err error, retryable bool) op.Result[T] {
	var be *core.BatchError
	if errors.As(err, &be) {
		result := op.Failed[T](err, false)
		if be.Applied == core.BatchAppliedPartial {
			result.Status = op.StatusPartial
		}
		return result
	}
	return op.Failed[T](err, retryable)
}
