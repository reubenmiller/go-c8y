package identity

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
)

// List returns all external identities for a managed object
func (s *Service) List(ctx context.Context, id string) op.Result[jsonmodels.Identity] {
	return core.ExecuteReturnCollection(ctx, s.ListB(id), "externalIds", "", jsonmodels.NewIdentity)
}

// Get retrieves a specific external identity
func (s *Service) Get(ctx context.Context, opts IdentityOptions) op.Result[jsonmodels.Identity] {
	return core.ExecuteReturnResult(ctx, s.GetB(opts), jsonmodels.NewIdentity)
}

// Create creates a new external identity for a managed object
func (s *Service) Create(ctx context.Context, id string, opts IdentityOptions) op.Result[jsonmodels.Identity] {
	return core.ExecuteReturnResult(ctx, s.CreateB(id, opts), jsonmodels.NewIdentity)
}

// Delete removes an external identity
func (s *Service) Delete(ctx context.Context, opts IdentityOptions) op.Result[jsonmodels.Identity] {
	return core.ExecuteReturnResult(ctx, s.DeleteB(opts), jsonmodels.NewIdentity)
}
