package managedobjects

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/source"
	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
)

// ResolveBodyRefs resolves the device references held at the given JSON paths
// of a pre-built request body (e.g. "source.id" of an event), so raw-body
// create calls accept the same references as the typed options. See
// source.ResolveBodyRefs. Lookups use the resolution context, so they run even
// under dry run or deferred execution.
func (r *DeviceResolver) ResolveBodyRefs(ctx context.Context, body any, paths ...string) (any, error) {
	if r == nil {
		return body, nil
	}
	return source.ResolveBodyRefs(ctx, body, func(ctx context.Context, ref string) (string, error) {
		return r.ResolveID(ctxhelpers.ResolutionContext(ctx), DeviceRef(ref), nil)
	}, paths...)
}
