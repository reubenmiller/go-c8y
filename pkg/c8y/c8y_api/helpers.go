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

func ExecuteNoResult(ctx context.Context, req *resty.Request) (*resty.Response, error) {
	resp, err := req.
		SetContext(ctx).
		Funcs(NoAcceptHeader).
		Send()

	if err != nil {
		return resp, err
	}
	if resp.IsError() {
		return resp, err
	}

	return resp, nil
}

func UnmarshalJSON(req *core.Response, data any) error {
	dec := json.NewDecoder(req.Response.Body)
	dec.UseNumber()
	return dec.Decode(&data)
}
