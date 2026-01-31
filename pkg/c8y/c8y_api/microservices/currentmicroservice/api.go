package currentmicroservice

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"resty.dev/v3"
)

var (
	ApiApplication              = "/application/currentApplication"
	ApiApplicationSubscriptions = "/application/currentApplication/subscriptions"
	ApiApplicationSettings      = "/application/currentApplication/settings"
)

// Service
type Service core.Service

func NewService(common *core.Service) *Service {
	return (*Service)(common)
}

// Get the current microservice
func (s *Service) Get(ctx context.Context) op.Result[jsonmodels.Microservice] {
	return core.ExecuteReturnResult(ctx, s.GetB(), jsonmodels.NewMicroservice)
}

func (s *Service) GetB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

// Retrieve the subscribed users of the current application
func (s *Service) ListUsers(ctx context.Context) op.Result[jsonmodels.MicroserviceUser] {
	return core.ExecuteReturnCollection(ctx, s.ListUsersB(), "users", "", jsonmodels.NewMicroserviceUser)
}

func (s *Service) ListUsersB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiApplicationSubscriptions)
	return core.NewTryRequest(s.Client, req)
}

// ListSettings returns the current application settings
func (s *Service) ListSettings(ctx context.Context) op.Result[jsonmodels.MicroserviceSetting] {
	return core.ExecuteReturnCollection(ctx, s.ListSettingsB(), "", "", jsonmodels.NewMicroserviceSetting)
}

func (s *Service) ListSettingsB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiApplicationSettings)
	return core.NewTryRequest(s.Client, req)
}
