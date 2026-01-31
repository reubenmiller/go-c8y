package core

import (
	"context"
	"net/http"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsondoc"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"resty.dev/v3"
)

// Execute a request and return the typed response
func Execute[T any](ctx context.Context, req *TryRequest) (*T, *resty.Response, error) {
	result := new(T)
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetResult(result).
		Send())

	if err != nil {
		return nil, resp, err
	}

	return result, resp, nil
}

func ExecuteReturnResult[T any](ctx context.Context, req *TryRequest, fromBytes func([]byte) T) op.Result[T] {
	resp, err := ExecuteResponseOnly(ctx, req)
	if err != nil {
		return op.Failed[T](err, true)
	}
	if resp.StatusCode() == http.StatusCreated {
		return op.NewCreated(fromBytes(resp.Bytes()))
	}
	// TODO: Should it return different status for update, delete etc.?
	return op.Ok(fromBytes(resp.Bytes()))
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
	result := op.NewOK(fromBytes([]byte(arrayResult.Raw)))
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

// Execute a request which expects a binary response which allows the user to read the body
func ExecuteBinaryResponse(ctx context.Context, req *TryRequest) (*BinaryResponse, error) {
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetDoNotParseResponse(true).
		Send())

	if err != nil {
		return nil, err
	}

	return NewBinaryResponse(resp), nil
}

// Execute a request and return the typed response
func ExecuteResultOnly[T any](ctx context.Context, req *TryRequest) (*T, error) {
	result := new(T)
	req.SetDefaultAcceptHeader()
	_, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetResult(result).
		Send())

	if err != nil {
		return nil, err
	}

	return result, nil
}

func ExecuteResultMap[k string, T any](ctx context.Context, req *TryRequest) (map[k]T, error) {
	result := make(map[k]T)
	req.SetDefaultAcceptHeader()
	_, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetResult(&result).
		Send())

	if err != nil {
		return nil, err
	}

	return result, nil
}

func ExecuteResultsArrayOnly[T any](ctx context.Context, req *TryRequest) ([]T, error) {
	result := make([]T, 0)
	req.SetDefaultAcceptHeader()
	_, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetResult(&result).
		Send())

	if err != nil {
		return nil, err
	}
	return result, nil
}

func ExecuteUpsertResultOnly[T any](ctx context.Context, create *TryRequest, update *TryRequest) (*T, error) {
	result, err := ExecuteResultOnly[T](ctx, create)
	if err == nil {
		return result, nil
	}

	if !ErrHasStatus(err, http.StatusConflict) {
		return result, err
	}
	return ExecuteResultOnly[T](ctx, update)
}

// Execute a request and return the response as text
func ExecuteResultText(ctx context.Context, req *TryRequest) (string, error) {
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		Send())

	if err != nil {
		return "", err
	}
	return resp.String(), nil
}

func ExecuteResultBytes(ctx context.Context, req *TryRequest) ([]byte, error) {
	resp, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		Send())

	if err != nil {
		return nil, err
	}
	return resp.Bytes(), nil
}

// Execute a request that doesn't any result only if there was an error or not
func ExecuteNoResult(ctx context.Context, req *TryRequest) error {
	_, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		Send())
	return err
}
