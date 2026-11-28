package users

import (
	"context"
	"net/url"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/currentuser"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/users/devicepermissions"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	apiUsers              = "/user/{tenantId}/users"
	apiUser               = "/user/{tenantId}/users/{id}"
	apiUserTFA            = "/user/{tenantId}/users/{id}/tfa"
	apiUserTOTPRevoke     = "/user/{tenantId}/users/{id}/totpSecret/revoke"
	apiUserGroupsWithUser = "/user/{tenantId}/users/{id}/groups"
	apiUserByName         = "/user/{tenantId}/userByName/{username}"
	apiLogout             = "/user/logout"
	apiLogoutAllUsers     = "/user/logout/{tenantId}/allUsers"
	apiPasswordReset      = "/user/passwordReset"
)

var paramID = "id"
var paramUsername = "username"

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

const resultProperty = "users"

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:               *s,
		Current:           currentuser.NewService(s),
		DevicePermissions: devicepermissions.NewService(s),
	}
}

// Service provides api to manage users
type Service struct {
	svc httpcore.Service

	// Current is the current-user sub-resource (/user/currentUser). This is
	// the canonical name, matching Tenants.Current and Applications.Current.
	Current *currentuser.Service

	// DevicePermissions is the device permissions sub-resource.
	DevicePermissions *devicepermissions.Service
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
	return httpcore.ExecuteCollection(ctx, s.listB(opt), resultProperty, types.ResponseFieldStatistics, jsonmodels.NewUser)
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiUsers)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
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

	// WithCertificates includes the user's provisioned (trusted) certificates
	// as provisionedCertificates (serialNumber, notBefore, notAfter).
	// Note: Cumulocity omits the "enabled" field in this response.
	// withCertificates and provisionedCertificates are not in the public core
	// OpenAPI spec; they may change without notice.
	WithCertificates bool `url:"withCertificates,omitempty"`
}

// Get a user
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.User] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewUser)
}

func (s *Service) getB(opt GetOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(paramID, string(opt.ID)).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiUser)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type GetByUsernameOptions struct {
	Username string
	Tenant   string
}

// Get a user by username
func (s *Service) GetByUsername(ctx context.Context, opt GetByUsernameOptions) op.Result[jsonmodels.User] {
	return httpcore.Execute(ctx, s.getByUsernameB(opt), jsonmodels.NewUser)
}

func (s *Service) getByUsernameB(opt GetByUsernameOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(paramUsername, opt.Username).
		SetURL(apiUserByName)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Create a user
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.User] {
	return httpcore.Execute(ctx, s.createB(body), jsonmodels.NewUser)
}

func (s *Service) createB(body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(body).
		SetURL(apiUsers)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type UpdateOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// Update a user
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.User] {
	return httpcore.Execute(ctx, s.updateB(opt, body), jsonmodels.NewUser)
}

func (s *Service) updateB(opt UpdateOptions, body any) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetPathParam(paramID, string(opt.ID)).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetBody(body).
		SetURL(apiUser)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type DeleteOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// Delete a user
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.deleteB(opt)).IgnoreNotFound()
}

func (s *Service) deleteB(opt DeleteOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(paramID, string(opt.ID)).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetURL(apiUser)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	return httpcore.ExecuteCollection(ctx, s.listGroupsWithUserB(opt), "references.#.group", types.ResponseFieldStatistics, jsonmodels.NewUserGroup)
}

// ListGroupsWithUserAll returns an iterator that transparently pages through all
// groups containing the given user.
func (s *Service) ListGroupsWithUserAll(ctx context.Context, opts ListGroupsOptions) *pagination.Iterator[jsonmodels.UserGroup] {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.UserGroup] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.ListGroupsWithUser(ctx, o)
		},
		jsonmodels.NewUserGroup,
	)
}

func (s *Service) listGroupsWithUserB(opt ListGroupsOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(paramID, opt.UserID).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(apiUserGroupsWithUser)
	return httpcore.NewTryRequest(s.svc.Client, req, resultProperty)
}

// UserSelfLink returns the self link for a user reference, suitable for the body
// of an AssignUser (add-user-to-group) call. A value that already looks like a
// user self link (it contains the "/users/" segment) is returned unchanged, so a
// piped user's self passes through; a bare user id/name is turned into its
// canonical self link using the client base URL and the given tenant. An empty
// ref (or empty tenant when one is needed) yields "".
//
// A user's self link is deterministic from its id (a user id equals its
// userName) and tenant, so this needs no network round-trip and works under
// dry-run. The user-name segment is percent-encoded the way Cumulocity stores it
// (e.g. "@" -> "%40").
func (s *Service) UserSelfLink(tenant, ref string) string {
	if ref == "" {
		return ""
	}
	if strings.Contains(ref, "/users/") {
		return ref
	}
	if tenant == "" {
		return ""
	}
	path := strings.ReplaceAll(apiUsers, "{"+core.PathParamTenantID+"}", tenant)
	return strings.TrimRight(s.svc.Client.BaseURL(), "/") + path + "/" + url.QueryEscape(ref)
}

// UserID returns the user id (which equals the user name in Cumulocity) for a
// user reference, suitable for the {id} path segment of an unassign call. It
// accepts the forms the CLI may supply: a bare id/name (returned unchanged), a
// self link (".../users/<id>", reduced to its last segment, percent-decoded), or
// a user / user-reference document (the embedded user id or name is used). An
// empty ref yields "". Like UserSelfLink this needs no network round-trip.
func (s *Service) UserID(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "{") {
		doc := jsondoc.New([]byte(ref))
		for _, p := range []string{"id", "userName", "user.id", "user.userName"} {
			if v := doc.Get(p).String(); v != "" {
				return v
			}
		}
		return ref
	}
	if i := strings.LastIndex(ref, "/users/"); i >= 0 {
		seg := strings.Trim(ref[i+len("/users/"):], "/")
		if dec, err := url.PathUnescape(seg); err == nil {
			return dec
		}
		return seg
	}
	return ref
}

// Logout terminates the current user's session and invalidates platform access tokens.
// Requires an active cookie-based or OAI-Secure session.
func (s *Service) Logout(ctx context.Context) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.logoutB())
}

func (s *Service) logoutB() *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetURL(apiLogout)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// LogoutAllUsers terminates all token-based sessions for every user in the given tenant.
// Requires ROLE_USER_MANAGEMENT_ADMIN and must be the current tenant.
func (s *Service) LogoutAllUsers(ctx context.Context, tenantID string) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.logoutAllUsersB(tenantID))
}

func (s *Service) logoutAllUsersB(tenantID string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPost).
		SetPathParam(core.PathParamTenantID, tenantID).
		SetURL(apiLogoutAllUsers)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	return httpcore.ExecuteNoContent(ctx, s.resetPasswordB(opt))
}

func (s *Service) resetPasswordB(opt ResetPasswordOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetBody(opt).
		SetURL(apiPasswordReset)
	return httpcore.NewTryRequest(s.svc.Client, req).WithNoAuth()
}

type GetTFAOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// GetTFA retrieves the two-factor authentication settings for a specific user.
// Leave Tenant empty to use the current context tenant.
func (s *Service) GetTFA(ctx context.Context, opt GetTFAOptions) op.Result[jsonmodels.UserTFA] {
	return httpcore.Execute(ctx, s.getTFAB(opt), jsonmodels.NewUserTFA)
}

func (s *Service) getTFAB(opt GetTFAOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(paramID, string(opt.ID)).
		SetURL(apiUserTFA)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

type RevokeTOTPSecretOptions struct {
	ID     UserRef `url:"-"`
	Tenant string  `url:"-"`
}

// RevokeTOTPSecret revokes (deletes) a user's TOTP (two-factor authentication)
// secret, forcing them to set up TFA again on next login. This is used by an
// administrator when a user loses their TFA configuration or it is compromised.
// Leave Tenant empty to use the current context tenant. A 204 No Content
// response indicates success.
func (s *Service) RevokeTOTPSecret(ctx context.Context, opt RevokeTOTPSecretOptions) op.Result[core.NoContent] {
	return httpcore.ExecuteNoContent(ctx, s.revokeTOTPSecretB(opt))
}

func (s *Service) revokeTOTPSecretB(opt RevokeTOTPSecretOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(paramID, string(opt.ID)).
		SetURL(apiUserTOTPRevoke)
	return httpcore.NewTryRequest(s.svc.Client, req)
}
