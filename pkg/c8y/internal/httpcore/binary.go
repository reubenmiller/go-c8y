package httpcore

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"resty.dev/v3"
)

// NewBinaryResponse wraps a resty response whose body has not been parsed.
// The returned response uses resty's body, so closing it also releases the
// request context.
func NewBinaryResponse(r *resty.Response) *core.BinaryResponse {
	if r == nil || r.RawResponse == nil {
		return &core.BinaryResponse{}
	}
	resp := *r.RawResponse
	resp.Body = r.Body
	return &core.BinaryResponse{Response: &resp}
}
