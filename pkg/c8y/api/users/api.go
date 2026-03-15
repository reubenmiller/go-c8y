package users

import (
	"context"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/currentuser"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/groups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	ApiUsers              = "/user/{tenantId}/users"
	ApiUser               = "/user/{tenantId}/users/{id}"
	ApiUserTFA            = "/user/{tenantId}/users/{id}/tfa"
	ApiUserGroupsWithUser = "/user/{tenantId}/users/{id}/groups"
	ApiUserByName         = "/user/{tenantId}/userByName/{username}"
	ApiLogout             = "/user/logout"
	ApiLogoutAllUsers     = "/user/logout/{tenantId}/allUsers"
	ApiPasswordReset      = "/user/passwordReset"
)

var ParamID = "id"
var ParamUsername = "username"

// PasswordStrength represents the Cumulocity password-strength classification
// returned by the server and accepted by the password-reset API.
type PasswordStrength string

const (
	PasswordStrengthRed    PasswordStrength = "RED"
	PasswordStrengthYellow PasswordStrength = "YELLOW"
	PasswordStrengthGreen  PasswordStrength = "GREEN"
)

const (
	passwordLower  = "abcdefghijklmnopqrstuvwxyz"
	passwordUpper  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	passwordDigits = "0123456789"
)

// CalculatePasswordStrength returns the Cumulocity password-strength level for
// the given password. The classification mirrors the algorithm used by the
// official @c8y/client TypeScript library:
//
//   - GREEN  — all four character classes (lower, upper, digit, symbol) present
//     and length ≥ 8
//   - YELLOW — two or three character classes present and length ≥ 8
//   - RED    — fewer than two character classes OR length < 8
func CalculatePasswordStrength(password string) PasswordStrength {
	if len(password) < 8 {
		return PasswordStrengthRed
	}
	var classes int
	if strings.ContainsAny(password, passwordLower) {
		classes++
	}
	if strings.ContainsAny(password, passwordUpper) {
		classes++
	}
	if strings.ContainsAny(password, passwordDigits) {
		classes++
	}
	for _, c := range password {
		if !strings.ContainsRune(passwordLower+passwordUpper+passwordDigits, c) {
			classes++
			break
		}
	}
	switch {
	case classes >= 4:
		return PasswordStrengthGreen
	case classes >= 2:
		return PasswordStrengthYellow
	default:
		return PasswordStrengthRed
	}
}

const ResultProperty = "users"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:     *s,
		CurrentUser: currentuser.NewService(s),
		Groups:      groups.NewService(s),
	}
}

// Service provides api to manage users
type Service struct {
	core.Service

	CurrentUser *currentuser.Service
	Groups      *groups.Service
}

// ListOptions to filter the users by
type ListOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	Username string `url:"username,omitempty"`

	Groups []string `url:"groups,omitempty"`

	// Exact username
	Owner string `url:"owner,omitempty"`

	// OnlyDevices If set to "true", result will contain only users created during bootstrap process (starting with "device_"). If flag is absent (or false) the result will not contain "device_" users.
	OnlyDevices bool `url:"onlyDevices,omitempty"`

	// WithSubusersCount if set to "true", then each of returned users will contain additional field "subusersCount" - number of direct subusers (users with corresponding "owner").
	WithSubusersCount bool `url:"withSubusersCount,omitempty"`

	pagination.PaginationOptions
}

// UserIterator provides iteration over users
type UserIterator = pagination.Iterator[jsonmodels.User]

// Retrieve all users in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.User] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewUser)
}

// ListAll returns an iterator for all users
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *UserIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.User] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewUser,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUsers)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// UserRef is a typed reference to a user. Construct it using ByID, ByDeviceUser, or cast a
// variable string with UserRef(id) when needed.
type UserRef string

// ByID creates a direct-ID user reference. No resolution is performed;
// the provided id is used as-is in the API call.
func ByID(id string) UserRef { return UserRef(id) }

// ByDeviceUser creates a user reference for the bootstrapped device user account
// that Cumulocity creates during bulk registration. The convention is that devices
// registered with ID "abc123" get a user account named "device_abc123".
func ByDeviceUser(deviceID string) UserRef { return UserRef("device_" + deviceID) }

type GetOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// Get a user
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewUser)
}

func (s *Service) getB(opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamID, string(opt.ID)).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

type GetByUsernameOptions struct {
	Username string
	Tenant   string
}

// Get a user by username
func (s *Service) GetByUsername(ctx context.Context, opt GetByUsernameOptions) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.getByUsernameB(opt), jsonmodels.NewUser)
}

func (s *Service) getByUsernameB(opt GetByUsernameOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamUsername, opt.Username).
		SetURL(ApiUserByName)
	return core.NewTryRequest(s.Client, req)
}

// Create a user
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.createB(body), jsonmodels.NewUser)
}

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(ApiUsers)
	return core.NewTryRequest(s.Client, req)
}

type UpdateOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// Update a user
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.User] {
	return core.Execute(ctx, s.updateB(opt, body), jsonmodels.NewUser)
}

func (s *Service) updateB(opt UpdateOptions, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(ParamID, string(opt.ID)).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetBody(body).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

type DeleteOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// Delete a user
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.deleteB(opt)).IgnoreNotFound()
}

func (s *Service) deleteB(opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamID, string(opt.ID)).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetURL(ApiUser)
	return core.NewTryRequest(s.Client, req)
}

// ListGroupsOptions to filter the user groups which contain a given user
type ListGroupsOptions struct {
	// Defaults to the current tenant
	Tenant string `url:"-"`

	UserID string `url:"-"`

	pagination.PaginationOptions
}

// List groups that contain a given user in the tenant
func (s *Service) ListGroupsWithUser(ctx context.Context, opt ListGroupsOptions) op.Result[jsonmodels.UserGroup] {
	return core.ExecuteCollection(ctx, s.listGroupsWithUserB(opt), "references.#.group", types.ResponseFieldStatistics, jsonmodels.NewUserGroup)
}

func (s *Service) listGroupsWithUserB(opt ListGroupsOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamID, opt.UserID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiUserGroupsWithUser)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

// Logout terminates the current user's session and invalidates platform access tokens.
// Requires an active cookie-based or OAI-Secure session.
func (s *Service) Logout(ctx context.Context) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.logoutB())
}

func (s *Service) logoutB() *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetURL(ApiLogout)
	return core.NewTryRequest(s.Client, req)
}

// LogoutAllUsers terminates all token-based sessions for every user in the given tenant.
// Requires ROLE_USER_MANAGEMENT_ADMIN and must be the current tenant.
func (s *Service) LogoutAllUsers(ctx context.Context, tenantID string) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.logoutAllUsersB(tenantID))
}

func (s *Service) logoutAllUsersB(tenantID string) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetURL(ApiLogoutAllUsers)
	return core.NewTryRequest(s.Client, req)
}

// ResetPasswordOptions contains the data needed for a token-based forced
// password reset triggered during login (when the server returns a 401 with a
// non-empty "passwordresettoken" response header).
type ResetPasswordOptions struct {
	// Tenant is the tenant ID the user belongs to.
	// When empty the server infers it from the request domain.
	Tenant string `url:"tenantId,omitempty" json:"-"`

	// Token is the one-time reset token from the 401 "passwordresettoken" header.
	Token string `json:"token"`

	// Email is the user's email / login name (the value typed in the login form).
	Email string `json:"email"`

	// NewPassword is the replacement password chosen by the user.
	NewPassword string `json:"newPassword"`

	// PasswordStrength indicates the complexity of NewPassword. Use CalculatePasswordStrength
	// to derive this value; the server still requires it even though it is
	// deprecated in the upstream TypeScript client library.
	PasswordStrength PasswordStrength `json:"passwordStrength"`
}

// ResetPassword performs a token-based forced password reset.
// This is called during login when the server returns a 401 with a
// "passwordresettoken" response header, signalling that the user must change
// their password before they can proceed.
func (s *Service) ResetPassword(ctx context.Context, opt ResetPasswordOptions) op.Result[core.NoContent] {
	return core.ExecuteNoContent(ctx, s.resetPasswordB(opt))
}

func (s *Service) resetPasswordB(opt ResetPasswordOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetBody(opt).
		SetURL(ApiPasswordReset)
	return core.NewTryRequest(s.Client, req).WithNoAuth()
}

type GetTFAOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// GetTFA retrieves the two-factor authentication settings for a specific user.
// Leave Tenant empty to use the current context tenant.
func (s *Service) GetTFA(ctx context.Context, opt GetTFAOptions) op.Result[jsonmodels.UserTFA] {
	return core.Execute(ctx, s.getTFAB(opt), jsonmodels.NewUserTFA)
}

func (s *Service) getTFAB(opt GetTFAOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(ParamID, string(opt.ID)).
		SetURL(ApiUserTFA)
	return core.NewTryRequest(s.Client, req)
}
