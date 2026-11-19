package currentmicroservice

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
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
type Service struct{ svc httpcore.Service }

func NewService(common *httpcore.Service) *Service {
	return &Service{svc: *common}
}

// Get the current microservice
func (s *Service) Get(ctx context.Context) op.Result[jsonmodels.Microservice] {
	return httpcore.Execute(ctx, s.getB(), jsonmodels.NewMicroservice)
}

func (s *Service) getB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Retrieve the subscribed users of the current application
func (s *Service) ListUsers(ctx context.Context) op.Result[jsonmodels.MicroserviceUser] {
	return httpcore.ExecuteCollection(ctx, s.listUsersB(), "users", "", jsonmodels.NewMicroserviceUser)
}

func (s *Service) listUsersB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplicationSubscriptions)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListSettings returns the current application settings.
//
// Cumulocity answers with a flat key/value JSON object (for example
// {"key1":"v1","key2":"v2"}), not a collection, so the result is a single
// MicroserviceSettings document. Use Map() or Get(key) to read the values.
func (s *Service) ListSettings(ctx context.Context) op.Result[jsonmodels.MicroserviceSettings] {
	return httpcore.Execute(ctx, s.listSettingsB(), jsonmodels.NewMicroserviceSettings)
}

func (s *Service) listSettingsB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiApplicationSettings)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
