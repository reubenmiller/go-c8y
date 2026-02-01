package core

import (
	"context"
	"net/http"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
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

func ExecuteReturnResult[T any](ctx context.Context, req *TryRequest, fromBytes func([]byte) T) op.Result[T] {
	resp, err := ExecuteResponseOnly(ctx, req)
	if err != nil {
		return op.Failed[T](err, true)
	}
	if resp.StatusCode() == http.StatusCreated {
		return op.Created(fromBytes(resp.Bytes()))
	}
	// TODO: Should it return different status for update, delete etc.?
	return op.OK(fromBytes(resp.Bytes()))
}

// ExecuteReturnCollection extracts an array from a collection response and puts metadata in Result.Meta
// arrayPath is the JSON path to the array (e.g., "managedObjects")
// metaPath is the JSON path to pagination metadata (e.g., "statistics")
func ExecuteReturnCollection[T any](ctx context.Context, req *TryRequest, arrayPath, metaPath string, fromBytes func([]byte) T) op.Result[T] {
	resp, err := ExecuteResponseOnly(ctx, req)
	if err != nil {
		return op.Failed[T](err, true)
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

	result.HTTPStatus = resp.StatusCode()
	result.RequestID = resp.Header().Get("X-Request-ID")

	return result
}

// Execute a request which expects a binary response which allows the user to read the body
func ExecuteBinaryResponse(ctx context.Context, req *TryRequest) op.Result[BinaryResponse] {
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Send())

	if err != nil {
		return op.Failed[BinaryResponse](err, false)
	}

	bin := NewBinaryResponse(resp)

	if resp.StatusCode() == http.StatusCreated {
		return op.Created(*bin)
	}

	if resp.StatusCode() == http.StatusOK {
		if req.Request.Method == http.MethodPut {
			return op.Updated(*bin)
		}
	}
	return op.OK(*bin)
}

type NoContent []byte

// Execute a request that doesn't any result only if there was an error or not
func ExecuteNoResult(ctx context.Context, req *TryRequest) op.Result[NoContent] {
	resp, err := ExecuteResponseOnly(ctx, req)

	meta := map[string]any{}
	meta["url"] = req.URL().String()
	meta["path"] = req.URL().Path

	if err != nil {
		return op.Failed[NoContent](err, true)
	}
	var empty NoContent
	if resp.StatusCode() == http.StatusNoContent {
		return op.NoContent(empty, meta)
	}
	// TODO: Should it return different status for update, delete etc.?
	return op.OK(empty, meta)
}
