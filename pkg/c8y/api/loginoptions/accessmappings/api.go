// Package accessmappings provides access to a login option's access mappings
// (/tenant/loginOptions/{typeOrId}/accessMappings), which map a condition to the
// applications and groups granted to a user on SSO login.
package accessmappings

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiAccessMappings = "/tenant/loginOptions/{typeOrId}/accessMappings"
var apiAccessMapping = "/tenant/loginOptions/{typeOrId}/accessMappings/{id}"

const (
	paramTypeOrID = "typeOrId"
	paramID       = "id"
)

const resultProperty = "accessMappings"

// Service provides access to a login option's access mappings.
type Service struct{ svc httpcore.Service }

func NewService(s *httpcore.Service) *Service {
	return &Service{svc: *s}
}

// List returns all access mappings for a login option (identified by type or id).
func (s *Service) List(ctx context.Context, typeOrID string) op.Result[jsonmodels.AccessMapping] {
	return httpcore.ExecuteCollection(ctx, s.listB(typeOrID), resultProperty, "", jsonmodels.NewAccessMapping)
}

func (s *Service) listB(typeOrID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetURL(apiAccessMappings)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// Get retrieves a single access mapping by id.
func (s *Service) Get(ctx context.Context, typeOrID, id string) op.Result[jsonmodels.AccessMapping] {
	return httpcore.Execute(ctx, s.getB(typeOrID, id), jsonmodels.NewAccessMapping)
}

func (s *Service) getB(typeOrID, id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetPathParam(paramID, id).
		SetURL(apiAccessMapping)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create adds a new access mapping to a login option.
func (s *Service) Create(ctx context.Context, typeOrID string, body any) op.Result[jsonmodels.AccessMapping] {
	return httpcore.Execute(ctx, s.createB(typeOrID, body), jsonmodels.NewAccessMapping)
}

func (s *Service) createB(typeOrID string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetBody(body).
		SetURL(apiAccessMappings)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update modifies an existing access mapping.
func (s *Service) Update(ctx context.Context, typeOrID, id string, body any) op.Result[jsonmodels.AccessMapping] {
	return httpcore.Execute(ctx, s.updateB(typeOrID, id, body), jsonmodels.NewAccessMapping)
}

func (s *Service) updateB(typeOrID, id string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetContentType(types.MimeTypeApplicationJSON).
		SetPathParam(paramTypeOrID, typeOrID).
		SetPathParam(paramID, id).
		SetBody(body).
		SetURL(apiAccessMapping)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Delete removes an access mapping.
func (s *Service) Delete(ctx context.Context, typeOrID, id string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(typeOrID, id))
}

func (s *Service) deleteB(typeOrID, id string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramTypeOrID, typeOrID).
		SetPathParam(paramID, id).
		SetURL(apiAccessMapping)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
