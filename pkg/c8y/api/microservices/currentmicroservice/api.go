package currentmicroservice

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	ApiApplication              = "/application/currentApplication"
	ApiApplicationSubscriptions = "/application/currentApplication/subscriptions"
	ApiApplicationSettings      = "/application/currentApplication/settings"
)

// Service
type Service struct{ core.Service }

func NewService(common *core.Service) *Service {
	return &Service{Service: *common}
}

// Get the current microservice
func (s *Service) Get(ctx context.Context) op.Result[jsonmodels.Microservice] {
	return core.Execute(ctx, s.getB(), jsonmodels.NewMicroservice)
}

func (s *Service) getB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiApplication)
	return core.NewTryRequest(s.Client, req)
}

// Retrieve the subscribed users of the current application
func (s *Service) ListUsers(ctx context.Context) op.Result[jsonmodels.MicroserviceUser] {
	return core.ExecuteCollection(ctx, s.listUsersB(), "users", "", jsonmodels.NewMicroserviceUser)
}

func (s *Service) listUsersB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiApplicationSubscriptions)
	return core.NewTryRequest(s.Client, req)
}

// ListSettings returns the current application settings
func (s *Service) ListSettings(ctx context.Context) op.Result[jsonmodels.MicroserviceSetting] {
	return core.ExecuteCollection(ctx, s.listSettingsB(), "", "", jsonmodels.NewMicroserviceSetting)
}

func (s *Service) listSettingsB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetURL(ApiApplicationSettings)
	return core.NewTryRequest(s.Client, req)
}
