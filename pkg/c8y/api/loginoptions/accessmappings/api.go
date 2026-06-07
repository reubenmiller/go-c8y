// Package accessmappings provides access to a login option's access mappings
// (/tenant/loginOptions/{typeOrId}/accessMappings), which map a condition to the
// applications and groups granted to a user on SSO login.
package accessmappings

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiAccessMappings = "/tenant/loginOptions/{typeOrId}/accessMappings"
var ApiAccessMapping = "/tenant/loginOptions/{typeOrId}/accessMappings/{id}"

const (
	ParamTypeOrID = "typeOrId"
	ParamID       = "id"
)

const ResultProperty = "accessMappings"

// Service provides access to a login option's access mappings.
type Service struct{ core.Service }

func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// List returns all access mappings for a login option (identified by type or id).
func (s *Service) List(ctx context.Context, typeOrID string) op.Result[jsonmodels.AccessMapping] {
	return core.ExecuteCollection(ctx, s.listB(typeOrID), ResultProperty, "", jsonmodels.NewAccessMapping)
}

func (s *Service) listB(typeOrID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetURL(ApiAccessMappings)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get retrieves a single access mapping by id.
func (s *Service) Get(ctx context.Context, typeOrID, id string) op.Result[jsonmodels.AccessMapping] {
	return core.Execute(ctx, s.getB(typeOrID, id), jsonmodels.NewAccessMapping)
}

func (s *Service) getB(typeOrID, id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetPathParam(ParamID, id).
		SetURL(ApiAccessMapping)
	return core.NewTryRequest(s.Client, req)
}

// Create adds a new access mapping to a login option.
func (s *Service) Create(ctx context.Context, typeOrID string, body any) op.Result[jsonmodels.AccessMapping] {
	return core.Execute(ctx, s.createB(typeOrID, body), jsonmodels.NewAccessMapping)
}

func (s *Service) createB(typeOrID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetBody(body).
		SetURL(ApiAccessMappings)
	return core.NewTryRequest(s.Client, req)
}

// Update modifies an existing access mapping.
func (s *Service) Update(ctx context.Context, typeOrID, id string, body any) op.Result[jsonmodels.AccessMapping] {
	return core.Execute(ctx, s.updateB(typeOrID, id, body), jsonmodels.NewAccessMapping)
}

func (s *Service) updateB(typeOrID, id string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetPathParam(ParamID, id).
		SetBody(body).
		SetURL(ApiAccessMapping)
	return core.NewTryRequest(s.Client, req)
}

// Delete removes an access mapping.
func (s *Service) Delete(ctx context.Context, typeOrID, id string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(typeOrID, id))
}

func (s *Service) deleteB(typeOrID, id string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamTypeOrID, typeOrID).
		SetPathParam(ParamID, id).
		SetURL(ApiAccessMapping)
	return core.NewTryRequest(s.Client, req)
}
