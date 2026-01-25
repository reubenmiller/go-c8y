package currentuser

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiCurrentUser = "/user/currentUser"
var ApiCurrentUserPassword = "/user/currentUser/password"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
	}
}

// Service provides api to manage the current user
type Service struct {
	core.Service
}

// Get the current user
func (s *Service) Get(ctx context.Context) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.GetB())
}

func (s *Service) GetB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiCurrentUser)
	return core.NewTryRequest(s.Client, req)
}

// Update the current user
func (s *Service) Update(ctx context.Context, body any) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.UpdateB(body))
}

func (s *Service) UpdateB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiCurrentUser)
	return core.NewTryRequest(s.Client, req)
}

type UpdatePasswordOptions struct {
	// The current password of the user performing the request
	CurrentUserPassword string `json:"currentUserPassword,omitempty"`

	// The new password to be set for the user performing the request
	NewPassword string `json:"newPassword,omitempty"`
}

// Update the current user's password
func (s *Service) UpdatePassword(ctx context.Context, body UpdatePasswordOptions) (*model.User, error) {
	return core.ExecuteResultOnly[model.User](ctx, s.UpdatePasswordB(body))
}

func (s *Service) UpdatePasswordB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiCurrentUserPassword)
	return core.NewTryRequest(s.Client, req)
}
