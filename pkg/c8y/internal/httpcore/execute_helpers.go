package httpcore

import (
	"context"
	"errors"
	"maps"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// deferred builds the request in dry-run mode, so all parameters (including
// resolved IDs) can be inspected, and returns a result that runs exec later.
func deferred[T any](ctx context.Context, req *TryRequest, doNotParse bool, meta map[string]any, exec func(context.Context) op.Result[T]) op.Result[T] {
	dryRunCtx := ctxhelpers.WithDryRun(ctx, true)
	req.ApplyContextRequestOptions(dryRunCtx)
	r := req.Request.SetContext(dryRunCtx)
	if doNotParse {
		r = r.SetResponseDoNotParse(true)
	}
	resp, _ := r.Send()

	var httpReq *http.Request
	if resp != nil {
		httpReq = resp.Request.RawRequest
	}
	result := op.Result[T]{Request: httpReq}
	mergeMeta(&result, meta)
	return op.WithExecutor(result, exec)
}

// dryRunRequest returns the sent request in dry-run mode, for inspection.
func dryRunRequest(ctx context.Context, resp *resty.Response) *http.Request {
	if resp != nil && ctxhelpers.IsDryRun(ctx) {
		return resp.Request.RawRequest
	}
	return nil
}

// DryRunReason is the "reason" metadata of a result for a request which was
// not sent because of dry run.
const DryRunReason = "dry run: request not sent"

// isPureDryRun reports whether the request was built but neither sent nor
// answered with mock data: the transport then returns a synthetic empty
// response which must not be reported as the outcome of the operation.
func isPureDryRun(ctx context.Context) bool {
	return ctxhelpers.IsDryRun(ctx) && !ctxhelpers.IsMockResponses(ctx)
}

// dryRunResult builds the result of a request which was not sent: Status
// Skipped, no error, the request in Request and the reason in Meta.
func dryRunResult[T any](ctx context.Context, data T, resp *resty.Response, meta map[string]any) op.Result[T] {
	result := op.Skipped(data, DryRunReason)
	result.Request = dryRunRequest(ctx, resp)
	if resp != nil {
		result.Duration = resp.Duration()
	}
	mergeMeta(&result, meta)
	return result
}

// firstMeta returns the optional extra metadata argument.
func firstMeta(extraMeta []map[string]any) map[string]any {
	if len(extraMeta) > 0 {
		return extraMeta[0]
	}
	return nil
}

// mergeMeta copies meta into the result's metadata.
func mergeMeta[T any](result *op.Result[T], meta map[string]any) {
	if meta == nil {
		return
	}
	if result.Meta == nil {
		result.Meta = make(map[string]any)
	}
	maps.Copy(result.Meta, meta)
}

// failedWithStatus builds a failed result with the HTTP status and duration
// taken from the response, or from the error when there is no response.
func failedWithStatus[T any](err error, resp *resty.Response) op.Result[T] {
	result := failed[T](err, true)
	if resp != nil {
		return op.WithHTTPStatus(op.WithDuration(result, resp.Duration()), resp.StatusCode())
	}
	var apiErr *core.Error
	if errors.As(err, &apiErr) && apiErr.Code >= 100 && apiErr.Code < 600 {
		result = op.WithHTTPStatus(result, apiErr.Code)
		if apiErr.Duration > 0 {
			result = op.WithDuration(result, apiErr.Duration)
		}
	}
	return result
}

// addCacheMeta records cache-related response headers in the metadata.
func addCacheMeta[T any](result *op.Result[T], resp *resty.Response) {
	for header, key := range map[string]string{"X-Cache": "x-cache", "X-From-Cache": "x-from-cache", "Age": "age"} {
		if v := resp.Header().Get(header); v != "" {
			if result.Meta == nil {
				result.Meta = make(map[string]any)
			}
			result.Meta[key] = v
		}
	}
}

// keepIgnorable marks a failed result so that IgnoreNotFound and
// IgnoreConflict keep their error when the caller asked for it with
// api.WithNotFoundAsError or api.WithConflictAsError.
func keepIgnorable[T any](ctx context.Context, r op.Result[T]) op.Result[T] {
	if ctxhelpers.NotFoundAsError(ctx) {
		r = op.WithKeepNotFound(r, true)
	}
	if ctxhelpers.ConflictAsError(ctx) {
		r = op.WithKeepConflict(r, true)
	}
	return r
}
