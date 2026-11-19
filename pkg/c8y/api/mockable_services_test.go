package api

import (
	"context"
	"testing"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
)

// Service methods stay non-generic so users can wrap services in their own
// interfaces for mocking.
type deviceGetter interface {
	Get(ctx context.Context, id string, opts managedobjects.GetOptions) op.Result[jsonmodels.ManagedObject]
}

func Test_ServicesCanBeMocked(t *testing.T) {
	var _ deviceGetter = (&Client{}).ManagedObjects
}
