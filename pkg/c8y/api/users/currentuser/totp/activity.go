package totp

import (
	"context"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

// GetActivity returns whether two-factor authentication with TOTP is active
// for the current user (GET /user/currentUser/totpSecret/activity).
func (s *Service) GetActivity(ctx context.Context) op.Result[jsonmodels.TOTPSecret] {
	return httpcore.Execute(ctx, s.getActivityB(), jsonmodels.NewTOTPSecret)
}

func (s *Service) getActivityB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTOTPSecretActivity)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
