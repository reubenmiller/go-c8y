package retentionrules

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiRetentionRules = "/retention/retentions"
var ApiRetentionRule = "/retention/retentions/{id}"

var ParamId = "id"

const ResultProperty = "retentionRules"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to managed retention rules
type Service struct {
	core.Service
}

// ListOptions to filter the retention rules by
type ListOptions struct {
	pagination.PaginationOptions
}

// Retrieve all login options available in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) (*model.RetentionRuleCollection, error) {
	return core.ExecuteResultOnly[model.RetentionRuleCollection](ctx, s.ListB(opt))
}

func (s *Service) ListB(opt any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiRetentionRules)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Get a retention rule
func (s *Service) Get(ctx context.Context, ID string) (*model.RetentionRule, error) {
	return core.ExecuteResultOnly[model.RetentionRule](ctx, s.GetB(ID))
}

func (s *Service) GetB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}

// Create a retention rule
func (s *Service) Create(ctx context.Context, body any) (*model.RetentionRule, error) {
	return core.ExecuteResultOnly[model.RetentionRule](ctx, s.CreateB(body))
}

func (s *Service) CreateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiRetentionRules)
	return core.NewTryRequest(s.Client, req)
}

// Update a retention rule
func (s *Service) Update(ctx context.Context, ID string, body any) (*model.LoginOption, error) {
	return core.ExecuteResultOnly[model.LoginOption](ctx, s.UpdateB(ID, body))
}

func (s *Service) UpdateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}

// Delete a retention rule
func (s *Service) Delete(ctx context.Context, ID string) error {
	return core.ExecuteNoResult(ctx, s.DeleteB(ID))
}

func (s *Service) DeleteB(ID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetURL(ApiRetentionRule)
	return core.NewTryRequest(s.Client, req)
}
