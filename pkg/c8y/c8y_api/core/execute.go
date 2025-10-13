package core

import (
	"context"

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

// Execute a request that doesn't any result only if there was an error or not
func ExecuteNoResult(ctx context.Context, req *TryRequest) error {
	_, err := coupleAPIErrors(req.Request.
		SetContext(ctx).
		Send())
	return err
}
