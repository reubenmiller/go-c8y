package dtm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Errors returned by DTM writes. They wrap the server error, so
// core.ErrHasStatus and core.IsNotFound keep working on them.
var (
	// ErrDefinitionInUse is returned when a definition cannot be deleted
	// because another definition still references it (HTTP 409).
	ErrDefinitionInUse = errors.New("definition is still referenced")

	// ErrDuplicateTitle is returned when a definition with a new identifier
	// uses a title that another definition already has (HTTP 409).
	ErrDuplicateTitle = errors.New("definition title is already used")

	// ErrReadOnlyDefinition is returned by Upsert when an existing
	// definition differs from the desired state but cannot be changed by
	// the current user (shared from the enterprise tenant, or a library
	// definition).
	ErrReadOnlyDefinition = errors.New("definition is read-only")
)

// skippedFrom turns r into a StatusSkipped result with no error, keeping the
// request, HTTP status and metadata for inspection.
func skippedFrom[T any](r op.Result[T], reason string) op.Result[T] {
	s := op.Skipped(r.Data, reason)
	s.HTTPStatus = r.HTTPStatus
	s.Request = r.Request
	s.Duration = r.Duration
	for k, v := range r.Meta {
		if _, exists := s.Meta[k]; !exists {
			s.Meta[k] = v
		}
	}
	return s
}

// skipIfNotFound maps a 404 error to StatusSkipped. Use it for deletes:
// a missing resource is the desired state.
func skipIfNotFound[T any](r op.Result[T]) op.Result[T] {
	if core.IsNotFound(r.Err) {
		return skippedFrom(r, "not found")
	}
	return r
}

// isNotAChild reports whether err is the 422 the DTM returns when removing a
// sub-asset or device that is not assigned to the asset.
func isNotAChild(err error) bool {
	return core.ErrHasStatus(err, http.StatusUnprocessableEntity) &&
		strings.Contains(err.Error(), "is not a child")
}

// toMap converts a request body (map, struct, or anything that marshals to a
// JSON object) to a map, merging annotations into it.
func toMap(body any, annotations ...model.Fragment) (map[string]any, error) {
	if body == nil {
		body = map[string]any{}
	}
	m, err := model.ApplyFragments(body, annotations)
	if err != nil {
		return nil, fmt.Errorf("body must marshal to a JSON object: %w", err)
	}
	return m, nil
}

// deepMerge returns base with overlay merged into it. Objects are merged key
// by key; any other overlay value replaces the base value. Neither input is
// modified.
func deepMerge(base, overlay map[string]any) map[string]any {
	out := make(map[string]any, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		bm, bok := out[k].(map[string]any)
		om, ook := v.(map[string]any)
		if bok && ook {
			out[k] = deepMerge(bm, om)
			continue
		}
		out[k] = v
	}
	return out
}

// jsonObject parses a JSON object, returning nil for anything else.
func jsonObject(b []byte) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return nil
	}
	return m
}

// deferrable runs fn now, or defers it under deferred execution, and tags
// the result with the operation name.
func deferrable[T any](ctx context.Context, operation string, fn func(context.Context) op.Result[T]) op.Result[T] {
	return op.ExecuteOrDefer(ctx, op.WithMeta(op.WithExecutor(op.Result[T]{}, fn), "operation", operation))
}
