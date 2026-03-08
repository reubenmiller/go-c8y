package identity

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
)

// List returns all external identities for a managed object
func (s *Service) List(ctx context.Context, id string) op.Result[jsonmodels.Identity] {
	return core.ExecuteCollection(ctx, s.listB(id), "externalIds", "", jsonmodels.NewIdentity)
}

// Get retrieves a specific external identity
func (s *Service) Get(ctx context.Context, opts IdentityOptions) op.Result[jsonmodels.Identity] {
	return core.Execute(ctx, s.getB(opts), jsonmodels.NewIdentity)
}

// Create creates a new external identity for a managed object
func (s *Service) Create(ctx context.Context, id string, opts IdentityOptions) op.Result[jsonmodels.Identity] {
	return core.Execute(ctx, s.createB(id, opts), jsonmodels.NewIdentity).IgnoreConflict()
}

// Delete removes an external identity
func (s *Service) Delete(ctx context.Context, opts IdentityOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(opts)).IgnoreNotFound()
}
