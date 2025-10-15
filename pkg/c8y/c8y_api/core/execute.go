package core

import (
	"context"
	"net/http"

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
	_, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		SetResult(result).
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
	if err != nil {
		return nil, err
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
