package userroles

import (
	"context"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/userroles/usergroups"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/userroles/users"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var ApiRoles = "/user/roles"
var ApiRole = "/user/roles/{name}"

var ParamName = "name"

const ResultProperty = "roles"

func NewService(s *core.Service) *Service {
	return &Service{
		Service: *s,
		Groups:  usergroups.NewService(s),
		Users:   users.NewService(s),
	}
}

// Service provides api to manage user roles
type Service struct {
	core.Service

	Groups *usergroups.Service
	Users  *users.Service
}

// ListOptions to filter the user roles by
type ListOptions struct {
	pagination.PaginationOptions
}

// RoleIterator provides iteration over roles
type RoleIterator = pagination.Iterator[jsonmodels.Role]

// Retrieve all user roles in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Role] {
	return core.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRole)
}

// ListAll returns an iterator for all user roles
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RoleIterator {
	return pagination.Paginate(
		ctx,
		opts.PaginationOptions,
		func(pageOpts pagination.PaginationOptions) op.Result[jsonmodels.Role] {
			o := opts
			o.PaginationOptions = pageOpts
			return s.List(ctx, o)
		},
		jsonmodels.NewRole,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiRoles)
	return core.NewTryRequest(s.Client, req, ResultProperty)
}

type GetOption struct {
	Name string `url:"-"`
}

// Get a user role
func (s *Service) Get(ctx context.Context, opt GetOption) op.Result[jsonmodels.Role] {
	return core.Execute(ctx, s.getB(opt), jsonmodels.NewRole)
}

func (s *Service) getB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, opt.Name).
		SetURL(ApiRole)
	return core.NewTryRequest(s.Client, req)
}

// RoleSelfLink returns the self link for a role reference, suitable for the body
// of an AssignRole call. A value that already looks like a self link (it contains
// the "/roles/" segment) is returned unchanged, so a piped role's self passes
// through; a bare role id/name (e.g. "ROLE_ALARM_READ") is turned into its
// canonical self link using the client base URL. An empty ref yields "".
//
// A role's self link is deterministic from its id (a role id equals its name),
// so this needs no network round-trip and works under dry-run.
func (s *Service) RoleSelfLink(ref string) string {
	if ref == "" {
		return ""
	}
	if strings.Contains(ref, "/roles/") {
		return ref
	}
	return strings.TrimRight(s.Client.BaseURL(), "/") + ApiRoles + "/" + ref
}

// RoleID returns the role id (which equals the role name in Cumulocity) for a
// role reference, suitable for the {roleId} path segment of an unassign call. It
// accepts the forms the CLI may supply: a bare id/name (returned unchanged), a
// self link (".../roles/ROLE_X", reduced to its last segment), or a role /
// role-reference document (the embedded role id or name is used). An empty ref
// yields "". Like RoleSelfLink this needs no network round-trip.
func (s *Service) RoleID(ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if strings.HasPrefix(ref, "{") {
		doc := jsondoc.New([]byte(ref))
		for _, p := range []string{"id", "name", "role.id", "role.name"} {
			if v := doc.Get(p).String(); v != "" {
				return v
			}
		}
		return ref
	}
	if i := strings.LastIndex(ref, "/roles/"); i >= 0 {
		return strings.Trim(ref[i+len("/roles/"):], "/")
	}
	return ref
}
