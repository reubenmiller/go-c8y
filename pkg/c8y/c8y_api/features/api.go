package features

import (
	"context"

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
func (s *Service) List(ctx context.Context) (*FeatureCollect, error) {
	return core.ExecuteResultOnly[FeatureCollect](ctx, s.ListB())
}

func (s *Service) ListB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiFeatures)
	return core.NewTryRequest(s.Client, req)
}

// Get a feature
func (s *Service) Get(ctx context.Context, key string) (*model.LoginOption, error) {
	return core.ExecuteResultOnly[model.LoginOption](ctx, s.GetB(key))
}

func (s *Service) GetB(key string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamKey, key).
		SetURL(ApiFeature)
	return core.NewTryRequest(s.Client, req)
}

// Update a feature
func (s *Service) Update(ctx context.Context, key string, body any) (*model.Feature, error) {
	return core.ExecuteResultOnly[model.Feature](ctx, s.UpdateB(key, body))
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
func (s *Service) Enable(ctx context.Context, key string) (*model.Feature, error) {
	return core.ExecuteResultOnly[model.Feature](ctx, s.UpdateB(key, &model.Feature{
		Active: true,
	}))
}

// Disable a feature
func (s *Service) Disable(ctx context.Context, key string) (*model.Feature, error) {
	return core.ExecuteResultOnly[model.Feature](ctx, s.UpdateB(key, &model.Feature{
		Active: false,
	}))
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
