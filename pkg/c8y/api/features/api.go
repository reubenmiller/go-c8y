package features

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiFeatures = "/features"
var ApiFeature = "/features/{key}"

var ParamKey = "key"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

type FeatureCollect []model.Feature

// Service provides api to managed features in Cumulocity
type Service struct {
	core.Service
}

// Retrieve all the features for the current tenant
func (s *Service) List(ctx context.Context) op.Result[jsonmodels.Feature] {
	return core.ExecuteCollection(ctx, s.listB(), "", "", jsonmodels.NewFeature)
}

func (s *Service) listB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiFeatures)
	return core.NewTryRequest(s.Client, req)
}

// Get a feature
func (s *Service) Get(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.Execute(ctx, s.getB(key), jsonmodels.NewFeature)
}

func (s *Service) getB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}

// Update a feature
func (s *Service) Update(ctx context.Context, key string, body any) op.Result[jsonmodels.Feature] {
	return core.Execute(ctx, s.updateB(key, body), jsonmodels.NewFeature)
}

func (s *Service) updateB(key string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamKey, key).
		SetBody(body).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}

// Enable a feature
func (s *Service) Enable(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.Execute(ctx, s.updateB(key, &model.Feature{
		Active: true,
	}), jsonmodels.NewFeature)
}

// Disable a feature
func (s *Service) Disable(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.Execute(ctx, s.updateB(key, &model.Feature{
		Active: false,
	}), jsonmodels.NewFeature)
}

// Delete a feature override
func (s *Service) Delete(ctx context.Context, key string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(key))
}

func (s *Service) deleteB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}
