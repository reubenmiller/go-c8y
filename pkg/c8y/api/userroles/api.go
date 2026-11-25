package userroles

import (
	"context"
	"fmt"
	"strings"

	ctxhelpers "github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsondoc"
	"github.com/reubenmiller/go-c8y/v2/pkg/matcher"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/userroles/inventory"
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

func NewService(s *httpcore.Service) *Service {
	return &Service{
		svc:       *s,
		Groups:    usergroups.NewService(s),
		Users:     users.NewService(s),
		Inventory: inventoryroles.NewService(s),
	}
}

// Service provides api to manage user roles
type Service struct {
	svc    httpcore.Service
	Groups *usergroups.Service
	Users  *users.Service

	// Inventory is the inventory roles sub-resource. Client.InventoryRoles is
	// a deprecated alias for the same instance.
	Inventory *inventoryroles.Service
}

// ListOptions to filter the user roles by
type ListOptions struct {
	pagination.PaginationOptions
}

// RoleIterator provides iteration over roles
type RoleIterator = pagination.Iterator[jsonmodels.Role]

// Retrieve all user roles in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Role] {
	return httpcore.ExecuteCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRole)
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

func (s *Service) listB(opt ListOptions) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetQueryParamsFromValues(httpcore.QueryParameters(opt)).
		SetURL(ApiRoles)
	return httpcore.NewTryRequest(s.svc.Client, req, ResultProperty)
}

type GetOption struct {
	Name string `url:"-"`
}

// Get a user role
func (s *Service) Get(ctx context.Context, opt GetOption) op.Result[jsonmodels.Role] {
	return httpcore.Execute(ctx, s.getB(opt), jsonmodels.NewRole)
}

func (s *Service) getB(opt GetOption) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, opt.Name).
		SetURL(ApiRole)
	return httpcore.NewTryRequest(s.svc.Client, req)
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
	return strings.TrimRight(s.svc.Client.BaseURL(), "/") + ApiRoles + "/" + ref
}

// ResolveID resolves a role reference to the role id (which equals the role
// name in Cumulocity). A role id, a self link or a role document is reduced
// to the id like RoleID, without a request. A name with "*" or "?" wildcards
// (e.g. "ROLE_ALARM_*") is matched, ignoring case, against the tenant's roles
// and the first match is used; the roles are requested also in a dry run.
// If meta is not nil, it is given the role's "id" and "name" and the "source"
// of the id ("id" or "name").
func (s *Service) ResolveID(ctx context.Context, ref string, meta map[string]any) (string, error) {
	id := s.RoleID(ref)
	if id == "" {
		return "", fmt.Errorf("empty role reference")
	}
	if !strings.ContainsAny(id, "*?") {
		if meta != nil {
			meta["id"] = id
			meta["name"] = id
			meta["source"] = "id"
		}
		return id, nil
	}

	ctx = ctxhelpers.ResolutionContext(ctx)
	roles := s.ListAll(ctx, ListOptions{
		PaginationOptions: pagination.PaginationOptions{PageSize: 2000},
	})
	for role, err := range roles.Items() {
		if err != nil {
			return "", err
		}
		if found, _ := matcher.MatchWithWildcards(role.Name(), id); found {
			if meta != nil {
				meta["id"] = role.ID()
				meta["name"] = role.Name()
				meta["source"] = "name"
			}
			return role.ID(), nil
		}
	}
	return "", fmt.Errorf("role not found: %s", id)
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
