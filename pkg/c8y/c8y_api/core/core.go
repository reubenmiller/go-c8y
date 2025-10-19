package core

import (
	"context"
	"io"
	"net/url"

	"github.com/google/go-querystring/query"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

type Service struct {
	Client *resty.Client
}

type TryRequest struct {
	Request  *resty.Request
	Client   *resty.Client
	Property string
}

func NewTryRequest(client *resty.Client, req *resty.Request, prop ...string) *TryRequest {
	property := ""
	if len(prop) > 0 {
		property = prop[0]
	}
	return &TryRequest{
		Client:   client,
		Request:  req,
		Property: property,
	}
}

func (r *TryRequest) URL() *url.URL {
	if u, err := url.Parse(r.Request.URL); err == nil {
		return u
	}
	return &url.URL{}
}

func (r *TryRequest) SetContext(ctx context.Context) *TryRequest {
	r.Request.SetContext(ctx)
	return r
}
func (r *TryRequest) SetProcessingMode(mode string) *TryRequest {
	r.Request.SetHeader(types.HeaderProcessingMode, mode)
	return r
}
func (r *TryRequest) SetProcessingModePersistent() *TryRequest {
	r.Request.SetHeader(types.HeaderProcessingMode, types.ProcessingModePersistent)
	return r
}
func (r *TryRequest) SetProcessingModeTransient() *TryRequest {
	r.Request.SetHeader(types.HeaderProcessingMode, types.ProcessingModeTransient)
	return r
}

func (r *TryRequest) SetProcessingModeCEP() *TryRequest {
	r.Request.SetHeader(types.HeaderProcessingMode, types.ProcessingModeCEP)
	return r
}

func (r *TryRequest) SetProcessingModeQuiescent() *TryRequest {
	r.Request.SetHeader(types.HeaderProcessingMode, types.ProcessingModeQuiescent)
	return r
}

func (r *TryRequest) SetNoResponse() *TryRequest {
	r.Request.Header.Del("Accept")
	return r
}

func (r *TryRequest) SetResponseBodyUnlimitedReads(v bool) *TryRequest {
	r.Request.SetResponseBodyUnlimitedReads(v)
	return r
}

func (r *TryRequest) SetOutputFileName(file string) *TryRequest {
	r.Request.SetOutputFileName(file)
	return r
}

func (r *TryRequest) SetSaveResponse(v bool) *TryRequest {
	r.Request.SetSaveResponse(v)
	return r
}

func (r *TryRequest) Funcs(funcs ...resty.RequestFunc) *TryRequest {
	r.Request.Funcs(funcs...)
	return r
}

func (r *TryRequest) SetResult(v any) *TryRequest {
	r.Request.SetResult(v)
	return r
}

func closeq(v any) {
	if c, ok := v.(io.Closer); ok {
		silently(c.Close())
	}
}

func silently(_ ...any) {}

func (r *TryRequest) Send() (*Response, error) {
	resp, err := r.Request.Send()

	// close body quietly to cleanup results
	closeq(r.Request.Body)

	return &Response{
		Request:  r,
		Response: resp,
	}, err
}

func QueryParameters(opt any) url.Values {
	v, _ := query.Values(opt)
	return v
}

func NoAuthorization() resty.RequestFunc {
	return func(r *resty.Request) *resty.Request {
		r.Header.Del("Authorization")
		return r
	}
}
