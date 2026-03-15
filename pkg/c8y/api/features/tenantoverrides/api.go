package tenantoverrides

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiFeatureByTenant = "/features/{key}/by-tenant"
var ApiFeatureForTenant = "/features/{key}/by-tenant/{tenantId}"

const ParamKey = "key"

// featureToggleValue is the request body for setting a feature toggle override.
type featureToggleValue struct {
	Active bool `json:"active"`
}

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides API access to per-tenant feature toggle overrides.
type Service struct {
	core.Service
}

// List retrieves all per-tenant value overrides for a given feature key.
// Requires management tenant.
func (s *Service) List(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.ExecuteCollection(ctx, s.listB(key), "", "", jsonmodels.NewFeature)
}

func (s *Service) listB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeatureByTenant)
	return core.NewTryRequest(s.Client, req)
}

// Set sets the feature toggle override for the current tenant.
func (s *Service) Set(ctx context.Context, key string, active bool) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.setB(key, active))
}

func (s *Service) setB(key string, active bool) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamKey, key).
		SetBody(&featureToggleValue{Active: active}).
		SetURL(ApiFeatureByTenant)
	return core.NewTryRequest(s.Client, req)
}

// Enable enables the feature toggle for the current tenant.
func (s *Service) Enable(ctx context.Context, key string) op.Result[core.NoContent] {
	return s.Set(ctx, key, true)
}

// Disable disables the feature toggle for the current tenant.
func (s *Service) Disable(ctx context.Context, key string) op.Result[core.NoContent] {
	return s.Set(ctx, key, false)
}

// Delete removes the feature toggle override for the current tenant.
func (s *Service) Delete(ctx context.Context, key string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(key)).IgnoreNotFound()
}

func (s *Service) deleteB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeatureByTenant)
	return core.NewTryRequest(s.Client, req)
}

// SetForTenant sets the feature toggle override for a specific tenant.
// Requires management tenant.
func (s *Service) SetForTenant(ctx context.Context, key string, tenantID string, active bool) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.setForTenantB(key, tenantID, active))
}

func (s *Service) setForTenantB(key string, tenantID string, active bool) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamKey, key).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetBody(&featureToggleValue{Active: active}).
		SetURL(ApiFeatureForTenant)
	return core.NewTryRequest(s.Client, req)
}

// EnableForTenant enables the feature toggle for a specific tenant.
// Requires management tenant.
func (s *Service) EnableForTenant(ctx context.Context, key string, tenantID string) op.Result[core.NoContent] {
	return s.SetForTenant(ctx, key, tenantID, true)
}

// DisableForTenant disables the feature toggle for a specific tenant.
// Requires management tenant.
func (s *Service) DisableForTenant(ctx context.Context, key string, tenantID string) op.Result[core.NoContent] {
	return s.SetForTenant(ctx, key, tenantID, false)
}

// DeleteForTenant removes the feature toggle override for a specific tenant.
// Requires management tenant.
func (s *Service) DeleteForTenant(ctx context.Context, key string, tenantID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteForTenantB(key, tenantID)).IgnoreNotFound()
}

func (s *Service) deleteForTenantB(key string, tenantID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamKey, key).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetURL(ApiFeatureForTenant)
	return core.NewTryRequest(s.Client, req)
}
