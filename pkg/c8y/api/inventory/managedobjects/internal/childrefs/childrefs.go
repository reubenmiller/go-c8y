// Package childrefs holds request helpers shared by the child reference
// services (childDevices, childAssets, childAdditions).
package childrefs

import (
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"resty.dev/v3"
)

// WithReferences sets a reference collection body naming childIDs on req (a
// DELETE on a child collection) and returns the request for execution.
func WithReferences(svc httpcore.Service, req *resty.Request, idParam, parentID string, childIDs []string) *httpcore.TryRequest {
	req.SetMethodDeleteAllowPayload(true).
		SetContentType(types.MimeTypeManagedObjectReferenceCollection).
		SetBody(model.ToManagedObjectChildReferences(childIDs)).
		SetPathParam(idParam, parentID)
	return httpcore.NewTryRequest(svc.Client, req)
}
