package currentmicroservice

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
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
		SetHeader("Accept", types.MimeTypeApplicationJSON).
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
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplicationSubscriptions)
	return core.NewTryRequest(s.Client, req)
}

// ListSettings returns the current application settings.
//
// Cumulocity answers with a flat key/value JSON object (for example
// {"key1":"v1","key2":"v2"}), not a collection, so the result is a single
// MicroserviceSettings document. Use Map() or Get(key) to read the values.
func (s *Service) ListSettings(ctx context.Context) op.Result[jsonmodels.MicroserviceSettings] {
	return core.Execute(ctx, s.listSettingsB(), jsonmodels.NewMicroserviceSettings)
}

func (s *Service) listSettingsB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplicationSettings)
	return core.NewTryRequest(s.Client, req)
}
