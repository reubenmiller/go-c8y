// Package currentapplication provides access to the Cumulocity current
// application endpoints (/application/currentApplication). These only work when
// authenticating with application bootstrap credentials, not user credentials.
package currentapplication

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	apiApplication              = "/application/currentApplication"
	apiApplicationSubscriptions = "/application/currentApplication/subscriptions"
)

// SubscriptionsResultProperty is the gjson path to the subscribed users embedded
// in the applicationUserCollection response.
const SubscriptionsResultProperty = "users"

// Service to interact with the current application
type Service struct{ svc httpcore.Service }

func NewService(s *httpcore.Service) *Service {
	return &Service{svc: *s}
}

// Get the current application. Only works with application bootstrap credentials
// (not user credentials).
func (s *Service) Get(ctx context.Context) op.Result[jsonmodels.Application] {
	return httpcore.Execute(ctx, s.getB(), jsonmodels.NewApplication)
}

func (s *Service) getB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Update the current application. Requires authentication with the application
// bootstrap user.
func (s *Service) Update(ctx context.Context, body any) op.Result[jsonmodels.Application] {
	return httpcore.Execute(ctx, s.updateB(body), jsonmodels.NewApplication)
}

func (s *Service) updateB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetBody(body).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiApplication)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// ListSubscriptions retrieves the subscribed users of the current application.
// The applicationUserCollection response embeds them under "users"; it is not
// separately paginated.
func (s *Service) ListSubscriptions(ctx context.Context) op.Result[jsonmodels.ApplicationUser] {
	return httpcore.ExecuteCollection(ctx, s.listSubscriptionsB(), SubscriptionsResultProperty, "", jsonmodels.NewApplicationUser)
}

func (s *Service) listSubscriptionsB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(apiApplicationSubscriptions)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
