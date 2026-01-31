package features

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
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
	return core.ExecuteReturnCollection(ctx, s.ListB(), "", "", jsonmodels.NewFeature)
}

func (s *Service) ListB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiFeatures)
	return core.NewTryRequest(s.Client, req)
}

// Get a feature
func (s *Service) Get(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.ExecuteReturnResult(ctx, s.GetB(key), jsonmodels.NewFeature)
}

func (s *Service) GetB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}

// Update a feature
func (s *Service) Update(ctx context.Context, key string, body any) op.Result[jsonmodels.Feature] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(key, body), jsonmodels.NewFeature)
}

func (s *Service) UpdateB(key string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamKey, key).
		SetBody(body).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}

// Enable a feature
func (s *Service) Enable(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(key, &model.Feature{
		Active: true,
	}), jsonmodels.NewFeature)
}

// Disable a feature
func (s *Service) Disable(ctx context.Context, key string) op.Result[jsonmodels.Feature] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(key, &model.Feature{
		Active: false,
	}), jsonmodels.NewFeature)
}

// Delete a feature override
func (s *Service) Delete(ctx context.Context, key string) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(key))
}

func (s *Service) DeleteB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}
