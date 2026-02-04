package core

import (
	"context"
	"net/http"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	ctxhelpers "github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/internal/context"
	"resty.dev/v3"
)

// Execute a request and return the typed response
func ExecuteResponseOnly(ctx context.Context, req *TryRequest) (*resty.Response, error) {
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		Send())

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func Execute[T any](ctx context.Context, req *TryRequest, fromBytes func([]byte) T, extraMeta ...map[string]any) op.Result[T] {
	// Check if execution should be deferred
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Build the request to capture all parameters (including resolved IDs)
		// but don't send it yet
		dryRunCtx := ctxhelpers.WithDryRun(ctx, true)
		resp, _ := req.Request.SetContext(dryRunCtx).Send()

		var httpReq *http.Request
		if resp != nil {
			httpReq = resp.Request.RawRequest
		}

		// Return a result with the executor function
		result := op.Result[T]{
			Request: httpReq,
		}

		// Merge extra metadata into the deferred result so it's available for inspection
		if len(extraMeta) > 0 && extraMeta[0] != nil {
			if result.Meta == nil {
				result.Meta = make(map[string]any)
			}
			for k, v := range extraMeta[0] {
				result.Meta[k] = v
			}
		}

		return result.WithExecutor(func(execCtx context.Context) op.Result[T] {
			return Execute(execCtx, req, fromBytes, extraMeta...)
		})
	}

	resp, err := ExecuteResponseOnly(ctx, req)

	// Only capture request in dry run mode for inspection
	var httpReq *http.Request
	if resp != nil && ctxhelpers.IsDryRun(ctx) {
		httpReq = resp.Request.RawRequest
	}

	if err != nil {
		result := op.Failed[T](err, true)
		if resp != nil {
			result = result.WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode())
		}
		// Merge extra metadata
		if len(extraMeta) > 0 && extraMeta[0] != nil {
			if result.Meta == nil {
				result.Meta = make(map[string]any)
			}
			for k, v := range extraMeta[0] {
				result.Meta[k] = v
			}
		}
		return result.WithRequest(httpReq)
	}

	var result op.Result[T]
	if resp.StatusCode() == http.StatusCreated {
		result = op.Created(fromBytes(resp.Bytes())).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	} else {
		// TODO: Should it return different status for update, delete etc.?
		result = op.OK(fromBytes(resp.Bytes())).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	}

	// Merge extra metadata
	if len(extraMeta) > 0 && extraMeta[0] != nil {
		if result.Meta == nil {
			result.Meta = make(map[string]any)
		}
		for k, v := range extraMeta[0] {
			result.Meta[k] = v
		}
	}

	return result
}

// ExecuteCollection extracts an array from a collection response and puts metadata in Result.Meta
// arrayPath is the JSON path to the array (e.g., "managedObjects")
// metaPath is the JSON path to pagination metadata (e.g., "statistics")
func ExecuteCollection[T any](ctx context.Context, req *TryRequest, arrayPath, metaPath string, fromBytes func([]byte) T, extraMeta ...map[string]any) op.Result[T] {
	// Check if execution should be deferred
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Build the request to capture all parameters
		dryRunCtx := ctxhelpers.WithDryRun(ctx, true)
		resp, _ := req.Request.SetContext(dryRunCtx).Send()

		var httpReq *http.Request
		if resp != nil {
			httpReq = resp.Request.RawRequest
		}

		// Return a result with the executor function
		result := op.Result[T]{
			Request: httpReq,
		}

		// Merge extra metadata into the deferred result so it's available for inspection
		if len(extraMeta) > 0 && extraMeta[0] != nil {
			if result.Meta == nil {
				result.Meta = make(map[string]any)
			}
			for k, v := range extraMeta[0] {
				result.Meta[k] = v
			}
		}

		return result.WithExecutor(func(execCtx context.Context) op.Result[T] {
			return ExecuteCollection(execCtx, req, arrayPath, metaPath, fromBytes, extraMeta...)
		})
	}

	resp, err := ExecuteResponseOnly(ctx, req)

	// Only capture request in dry run mode for inspection
	var httpReq *http.Request
	if resp != nil && ctxhelpers.IsDryRun(ctx) {
		httpReq = resp.Request.RawRequest
	}

	if err != nil {
		result := op.Failed[T](err, true)
		if resp != nil {
			result = result.WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode())
		}
		return result.WithRequest(httpReq)
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
	result.Meta["totalPages"] = doc.Get("statistics.totalPages").Int()
	result.Meta["totalElements"] = doc.Get("statistics.totalElements").Int()

	// Merge extra metadata
	if len(extraMeta) > 0 && extraMeta[0] != nil {
		for k, v := range extraMeta[0] {
			result.Meta[k] = v
		}
	}

	result.HTTPStatus = resp.StatusCode()
	result.RequestID = resp.Header().Get("X-Request-ID")
	result.Duration = resp.Duration()
	result.Request = httpReq

	return result
}

// Execute a request which expects a binary response which allows the user to read the body
func ExecuteBinary(ctx context.Context, req *TryRequest) op.Result[BinaryResponse] {
	// Check if execution should be deferred
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Build the request to capture all parameters
		dryRunCtx := ctxhelpers.WithDryRun(ctx, true)
		resp, _ := req.Request.SetContext(dryRunCtx).SetDoNotParseResponse(true).Send()

		var httpReq *http.Request
		if resp != nil {
			httpReq = resp.Request.RawRequest
		}

		// Return a result with the executor function
		return op.Result[BinaryResponse]{
			Request: httpReq,
		}.WithExecutor(func(execCtx context.Context) op.Result[BinaryResponse] {
			return ExecuteBinary(execCtx, req)
		})
	}

	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Send())

	if err != nil {
		return op.Failed[BinaryResponse](err, false)
	}

	bin := NewBinaryResponse(resp)

	// Only capture request in dry run mode for inspection
	var httpReq *http.Request
	if resp != nil && ctxhelpers.IsDryRun(ctx) {
		httpReq = resp.Request.RawRequest
	}

	if resp.StatusCode() == http.StatusCreated {
		return op.Created(*bin).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	}

	if resp.StatusCode() == http.StatusOK {
		if req.Request.Method == http.MethodPut {
			return op.Updated(*bin).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
		}
	}
	return op.OK(*bin).WithDuration(resp.Duration()).WithRequest(httpReq)
}

type NoContent []byte

// Execute a request that doesn't any result only if there was an error or not
func ExecuteNoContent(ctx context.Context, req *TryRequest, extraMeta ...map[string]any) op.Result[NoContent] {
	// Check if execution should be deferred
	if ctxhelpers.IsDeferredExecution(ctx) {
		// Build the request to capture all parameters
		dryRunCtx := ctxhelpers.WithDryRun(ctx, true)
		resp, _ := req.Request.SetContext(dryRunCtx).Send()

		var httpReq *http.Request
		if resp != nil {
			httpReq = resp.Request.RawRequest
		}

		// Return a result with the executor function
		result := op.Result[NoContent]{
			Request: httpReq,
		}

		// Merge extra metadata into the deferred result so it's available for inspection
		if len(extraMeta) > 0 && extraMeta[0] != nil {
			if result.Meta == nil {
				result.Meta = make(map[string]any)
			}
			for k, v := range extraMeta[0] {
				result.Meta[k] = v
			}
		}

		return result.WithExecutor(func(execCtx context.Context) op.Result[NoContent] {
			return ExecuteNoContent(execCtx, req, extraMeta...)
		})
	}

	resp, err := ExecuteResponseOnly(ctx, req)

	meta := map[string]any{}
	meta["url"] = req.URL().String()
	meta["path"] = req.URL().Path

	// Merge extra metadata
	if len(extraMeta) > 0 && extraMeta[0] != nil {
		for k, v := range extraMeta[0] {
			meta[k] = v
		}
	}

	// Only capture request in dry run mode for inspection
	var httpReq *http.Request
	if resp != nil && ctxhelpers.IsDryRun(ctx) {
		httpReq = resp.Request.RawRequest
	}

	if err != nil {
		result := op.Failed[NoContent](err, true)
		if resp != nil {
			result = result.WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode())
		}
		return result.WithRequest(httpReq)
	}
	var empty NoContent
	if resp.StatusCode() == http.StatusNoContent {
		return op.NoContent(empty, meta).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
	}
	// TODO: Should it return different status for update, delete etc.?
	return op.OK(empty, meta).WithDuration(resp.Duration()).WithHTTPStatus(resp.StatusCode()).WithRequest(httpReq)
}
