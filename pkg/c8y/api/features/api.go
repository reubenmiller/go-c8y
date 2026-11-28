package features

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/features/tenantoverrides"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var apiFeatures = "/features"
var apiFeature = "/features/{key}"

// apiFeatureByTenant is the per-tenant feature-toggle override endpoint. The
// platform requires it for mutating a tenant's value (PUT/DELETE); the plain
// apiFeature path is read-only for the current tenant.
var apiFeatureByTenant = "/features/{key}/by-tenant"

var paramKey = "key"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:     *s,
		Tenants: tenantoverrides.NewService(s),
	}
}

type FeatureCollect []model.Feature

// Service provides api to managed features in Cumulocity
type Service struct {
	svc     httpcore.Service
	Tenants *tenantoverrides.Service
}

// Retrieve all the features for the current tenant
func (s *Service) List(ctx context.Context) op.Result[jsonmodels.Feature] {
	return httpcore.ExecuteCollection(ctx, s.listB(), "", "", jsonmodels.NewFeature)
}

func (s *Service) listB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiFeatures)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Get a feature
func (s *Service) Get(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return httpcore.Execute(ctx, s.getB(key), jsonmodels.NewFeature)
}

func (s *Service) getB(key string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramKey, key).
		SetURL(apiFeature)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update a feature
func (s *Service) Update(ctx context.Context, key string, body any) op.Result[jsonmodels.Feature] {
	return httpcore.Execute(ctx, s.updateB(key, body), jsonmodels.NewFeature)
}

func (s *Service) updateB(key string, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(paramKey, key).
		SetBody(body).
		SetURL(apiFeatureByTenant)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Enable a feature
func (s *Service) Enable(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return httpcore.Execute(ctx, s.updateB(key, &model.Feature{
		Active: true,
	}), jsonmodels.NewFeature)
}

// Disable a feature
func (s *Service) Disable(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return httpcore.Execute(ctx, s.updateB(key, &model.Feature{
		Active: false,
	}), jsonmodels.NewFeature)
}

// Delete removes the feature override for the current tenant.
func (s *Service) Delete(ctx context.Context, key string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(key)).IgnoreNotFound()
}

func (s *Service) deleteB(key string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramKey, key).
		SetURL(apiFeatureByTenant)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
