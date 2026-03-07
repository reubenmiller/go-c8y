// Package totp provides an API service for managing TOTP (Time-based One-Time
// Password) two-factor authentication for the current Cumulocity user.
package totp

import (
	"context"

	"github.com/reubenmiller/go-c8y/pkg/c8y/api/authentication"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	ApiTOTPSecret         = "/user/currentUser/totpSecret"
	ApiTOTPSecretVerify   = "/user/currentUser/totpSecret/verify"
	ApiTOTPSecretActivity = "/user/currentUser/totpSecret/activity"
)

// TOTPChallenge describes the context in which a TOTP code is being requested.
// It is passed to a TOTPCodeFunc so the caller can tailor the prompt message.
type TOTPChallenge struct {
	// IsSetup is true when the user is enrolling TOTP for the first time and
	// must enter a code to verify that their authenticator app is working.
	// It is false for a normal login challenge where TOTP is already active.
	IsSetup bool

	// Message is any hint text surfaced from the server error, lower-cased.
	Message string
}

// TOTPCodeFunc is called whenever the login flow needs a TOTP code from the
// user. Implementations should prompt interactively (terminal, UI dialog, …)
// and return the code as a string, or an error to abort the login.
type TOTPCodeFunc func(ctx context.Context, challenge TOTPChallenge) (string, error)

// NewService creates a new TOTP service.
func NewService(s *core.Service) *Service {
	return &Service{Service: *s}
}

// Service provides operations for managing TOTP two-factor authentication.
type Service struct {
	core.Service
}

// GenerateSecret requests a new TOTP secret for the current user.
//
// Cumulocity requires basic authentication for this endpoint; a bearer token
// is not accepted. Pass the user's tenant, username and password in auth.
func (s *Service) GenerateSecret(ctx context.Context) op.Result[jsonmodels.TOTPSecret] {
	return core.Execute(ctx, s.generateSecretB(), jsonmodels.NewTOTPSecret)
}

func (s *Service) generateSecretB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiTOTPSecret)
	return core.NewTryRequest(s.Client, req).
		WithAuthType(authentication.AuthTypeBasic)
}

// VerifyCode checks that the given TOTP code is valid.
//
// This is called after GenerateSecret to confirm the user has successfully
// enrolled their authenticator app before activating TOTP.
func (s *Service) VerifyCode(ctx context.Context, code string) op.Result[jsonmodels.TOTPSecret] {
	return core.Execute(ctx, s.verifyCodeB(code), jsonmodels.NewTOTPSecret)
}

func (s *Service) verifyCodeB(code string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(map[string]any{"code": code}).
		SetURL(ApiTOTPSecretVerify)
	return core.NewTryRequest(s.Client, req).WithAuthType(authentication.AuthTypeBasic)
}

// SetActivity activates or deactivates TOTP for the current user.
//
// Pass isActive=true to enable TOTP (call after VerifyCode succeeds), or
// isActive=false to disable it. Like GenerateSecret, this endpoint requires
// basic authentication.
func (s *Service) SetActivity(ctx context.Context, isActive bool) op.Result[jsonmodels.TOTPSecret] {
	return core.Execute(ctx, s.setActivityB(isActive), jsonmodels.NewTOTPSecret)
}

func (s *Service) setActivityB(isActive bool) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(map[string]any{"isActive": isActive}).
		SetURL(ApiTOTPSecretActivity)
	return core.NewTryRequest(s.Client, req).WithAuthType(authentication.AuthTypeBasic)
}
