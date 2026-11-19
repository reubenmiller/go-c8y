package childadditions

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// UnassignMany removes several child additions from a parent managed object in one
// request (DELETE …/ChildAdditions with a reference collection body).
func (s *Service) UnassignMany(ctx context.Context, parentID string, childIDs ...string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.unassignManyB(parentID, childIDs))
}

func (s *Service) unassignManyB(parentID string, childIDs []string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetMethodDeleteAllowPayload(true).
		SetContentType(types.MimeTypeManagedObjectReferenceCollection).
		SetBody(model.ToManagedObjectChildReferences(childIDs)).
		SetPathParam(ParamID, parentID).
		SetURL(ApiManagedObjectChildAdditions)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
