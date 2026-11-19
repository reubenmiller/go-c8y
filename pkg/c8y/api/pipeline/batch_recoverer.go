package pipeline

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
)

// BatchRecoverer completes a batch request that failed after part of it was
// written. items is the slice that was sent in the failed request. It returns
// all results in the order of items.
//
// Services that can recover a partial batch expose a method with this
// signature, for example managedobjects.Service.RecoverCreateBulk.
type BatchRecoverer[T any] func(ctx context.Context, items []any, err *core.BatchError) ([]T, error)
