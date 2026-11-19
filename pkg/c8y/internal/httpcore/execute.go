package httpcore

import (
	"context"
	"maps"
	"net/http"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// Execute a request and return the typed response
func ExecuteResponseOnly(ctx context.Context, req *TryRequest) (*resty.Response, error) {
	switch req.authType {
	case authentication.AuthTypeBasic:
		// Skip the token-source middleware and clear any bearer token that resty
		// copied from the client (c.authToken → r.AuthToken) so that addCredentials
		// only applies basic auth.
		ctx = ctxhelpers.WithSkipTokenSource(ctx)
		req.Request.SetAuthToken("")
	case authentication.AuthTypeNone:
		// Skip token source, clear bearer, and signal the transport to strip the
		// Authorization header entirely (prevents resty's own credentials from leaking).
		ctx = ctxhelpers.WithSkipTokenSource(ctx)
		req.Request.SetAuthToken("")
		ctx = ctxhelpers.WithNoAuth(ctx)
	}
	// Apply context-driven request options (e.g. processing mode) that live on
	// the TryRequest wrapper — resty's own SetContext below does not know them.
	req.ApplyContextRequestOptions(ctx)

	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		Send())

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func Execute[T any](ctx context.Context, req *TryRequest, fromBytes func([]byte) T, extraMeta ...map[string]any) op.Result[T] {
	meta := firstMeta(extraMeta)
	if ctxhelpers.IsDeferredExecution(ctx) {
		return deferred(ctx, req, false, meta, func(execCtx context.Context) op.Result[T] {
			return Execute(execCtx, req, fromBytes, extraMeta...)
		})
	}

	resp, err := ExecuteResponseOnly(ctx, req)
	httpReq := dryRunRequest(ctx, resp)

	if err != nil {
		result := failedWithStatus[T](err, resp)
		mergeMeta(&result, meta)
		return result.WithRequest(httpReq)
	}

	var result op.Result[T]
	switch {
	case resp.StatusCode() == http.StatusCreated:
		result = op.Created(fromBytes(resp.Bytes()))
	case resp.StatusCode() == http.StatusOK && (req.Request.Method == http.MethodPut || req.Request.Method == http.MethodPatch):
		result = op.Updated(fromBytes(resp.Bytes()))
	default:
		result = op.OK(fromBytes(resp.Bytes()))
	}
	result = result.WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	addCacheMeta(&result, resp)
	mergeMeta(&result, meta)
	return result
}

// ExecuteCollection extracts an array from a collection response and puts metadata in Result.Meta
// arrayPath is the JSON path to the array (e.g., "managedObjects")
// metaPath is the JSON path to pagination metadata (e.g., "statistics")
func ExecuteCollection[T any](ctx context.Context, req *TryRequest, arrayPath, metaPath string, fromBytes func([]byte) T, extraMeta ...map[string]any) op.Result[T] {
	if ctxhelpers.IsDeferredExecution(ctx) {
		return deferred(ctx, req, false, firstMeta(extraMeta), func(execCtx context.Context) op.Result[T] {
			return ExecuteCollection(execCtx, req, arrayPath, metaPath, fromBytes, extraMeta...)
		})
	}

	resp, err := ExecuteResponseOnly(ctx, req)
	httpReq := dryRunRequest(ctx, resp)

	if err != nil {
		return failedWithStatus[T](err, resp).WithRequest(httpReq)
	}

	// TODO: how to do this more efficiently
	doc := jsondoc.New(resp.Bytes())

	// Extract the array as the main data
	arrayResult := doc.Get(arrayPath)

	// Extract metadata
	result := op.OK(fromBytes([]byte(arrayResult.Raw)))
	if metaPath != "" {
		stats := doc.Get(metaPath)
		if stats.Exists() {
			result.Meta["currentPage"] = stats.Get("currentPage").Int()
			result.Meta["pageSize"] = stats.Get("pageSize").Int()
			if totalPages := stats.Get("totalPages"); totalPages.Exists() {
				result.Meta["totalPages"] = totalPages.Int()
			}
		}
	}
	// Pagination info
	result.Meta["next"] = doc.Get("next").String()
	result.Meta["self"] = doc.Get("self").String()
	result.Meta["prev"] = doc.Get("prev").String()

	result.Meta["currentPage"] = doc.Get("statistics.currentPage").Int()
	result.Meta["pageSize"] = doc.Get("statistics.pageSize").Int()
	// Totals are only present when requested (withTotalPages/withTotalElements).
	// Leave them unset otherwise: a zero totalPages would read as "last page".
	if totalPages := doc.Get("statistics.totalPages"); totalPages.Exists() {
		result.Meta["totalPages"] = totalPages.Int()
	}
	if totalElements := doc.Get("statistics.totalElements"); totalElements.Exists() {
		result.Meta["totalElements"] = totalElements.Int()
	}

	mergeMeta(&result, firstMeta(extraMeta))

	result.HTTPStatus = resp.StatusCode()
	result.RequestID = resp.Header().Get("X-Request-ID")
	result.Duration = resp.Duration()
	result.Request = httpReq
	// Retain the full envelope (un-plucked) for callers that want raw output.
	// doc.Raw() shares the buffer already parsed above, so this is zero-copy.
	result.Response = doc.Raw()

	addCacheMeta(&result, resp)
	return result
}

// Execute a request which expects a binary response which allows the user to read the body
func ExecuteBinary(ctx context.Context, req *TryRequest) op.Result[core.BinaryResponse] {
	if ctxhelpers.IsDeferredExecution(ctx) {
		return deferred(ctx, req, true, nil, func(execCtx context.Context) op.Result[core.BinaryResponse] {
			return ExecuteBinary(execCtx, req)
		})
	}

	req.ApplyContextRequestOptions(ctx)
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetResponseDoNotParse(true).
		Send())

	if err != nil {
		return op.Failed[core.BinaryResponse](err, false).WithHTTPStatus(getStatusCodeFromError(err))
	}

	bin := NewBinaryResponse(resp)

	httpReq := dryRunRequest(ctx, resp)

	if resp.StatusCode() == http.StatusCreated {
		return op.Created(*bin).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	}

	if resp.StatusCode() == http.StatusOK {
		if req.Request.Method == http.MethodPut {
			return op.Updated(*bin).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
		}
	}
	return op.OK(*bin).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
}

// Execute a request that doesn't any result only if there was an error or not
func ExecuteNoContent(ctx context.Context, req *TryRequest, extraMeta ...map[string]any) op.Result[core.NoContent] {
	if ctxhelpers.IsDeferredExecution(ctx) {
		return deferred(ctx, req, false, firstMeta(extraMeta), func(execCtx context.Context) op.Result[core.NoContent] {
			return ExecuteNoContent(execCtx, req, extraMeta...)
		})
	}

	resp, err := ExecuteResponseOnly(ctx, req)

	meta := map[string]any{
		"url":  req.URL().String(),
		"path": req.URL().Path,
	}
	maps.Copy(meta, firstMeta(extraMeta))
	httpReq := dryRunRequest(ctx, resp)

	if err != nil {
		// The response is nil on an HTTP error, so take the status from the
		// error. Without it, IgnoreNotFound() and IsNotFound() on a Delete
		// result would never see the request's own 404.
		result := failed[core.NoContent](err, true).WithHTTPStatus(getStatusCodeFromError(err))
		if resp != nil {
			result = result.WithDuration(resp.Duration())
			if code := resp.StatusCode(); code != 0 {
				result = result.WithHTTPStatus(code)
			}
		}
		return result.WithRequest(httpReq)
	}
	var empty core.NoContent
	if resp.StatusCode() == http.StatusNoContent {
		return op.NoContent(empty, meta).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	}
	if resp.StatusCode() == http.StatusOK && (req.Request.Method == http.MethodPut || req.Request.Method == http.MethodPatch) {
		return op.Updated(empty, meta).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	}
	return op.OK(empty, meta).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
}
