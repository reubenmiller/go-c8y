package childdevices

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects/internal/childrefs"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// UnassignMany removes several child devices from a parent managed object in one
// request (DELETE …/ChildDevices with a reference collection body).
func (s *Service) UnassignMany(ctx context.Context, parentID string, childIDs ...string) op.Result[core.NoContent] {
	req := s.svc.Client.R().SetMethod(resty.MethodDelete).SetURL(apiManagedObjectChildDevices)
	return httpcore.ExecuteNoContent(ctx, childrefs.WithReferences(s.svc, req, paramID, parentID, childIDs))
}
