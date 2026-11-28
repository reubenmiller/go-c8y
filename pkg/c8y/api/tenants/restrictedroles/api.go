// Package restrictedroles manages the roles that microservices subscribed to
// or owned by a tenant may not request.
package restrictedroles

import (
	"context"
	"encoding/json"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/internal/httpcore"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

var (
	apiRestrictedRoles = "/tenant/tenants/{tenantId}/applications/restricted-roles"
	apiRestrictedRole  = "/tenant/tenants/{tenantId}/applications/restricted-roles/{roleId}"
)

// paramRoleID is the role path parameter.
const paramRoleID = "roleId"

// Options selects the tenant. Tenant defaults to the client's tenant.
type Options struct {
	Tenant string
}

// Service manages a tenant's restricted microservice roles.
type Service struct {
	svc httpcore.Service
}

func NewService(s *httpcore.Service) *Service {
	return &Service{svc: *s}
}

// List returns the roles microservices of the tenant may not request.
func (s *Service) List(ctx context.Context, opt Options) op.Result[[]string] {
	return httpcore.Execute(ctx, s.listB(opt), parseRoles)
}

func (s *Service) listB(opt Options) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetURL(apiRestrictedRoles)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Replace sets the full list of restricted roles.
func (s *Service) Replace(ctx context.Context, opt Options, roles []string) op.Result[jsonmodels.Tenant] {
	return httpcore.Execute(ctx, s.replaceB(opt, roles), jsonmodels.NewTenant)
}

func (s *Service) replaceB(opt Options, roles []string) *httpcore.TryRequest {
	if roles == nil {
		roles = []string{}
	}
	req := s.svc.Client.R().
		SetMethod(resty.MethodPut).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetHeader("Content-Type", types.MimeTypeApplicationJSON).
		SetBody(roles).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetURL(apiRestrictedRoles)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

// Add adds one role to the restricted roles.
func (s *Service) Add(ctx context.Context, opt Options, role string) op.Result[jsonmodels.Tenant] {
	return httpcore.Execute(ctx, s.roleB(resty.MethodPost, opt, role), jsonmodels.NewTenant)
}

// Remove removes one role from the restricted roles.
func (s *Service) Remove(ctx context.Context, opt Options, role string) op.Result[jsonmodels.Tenant] {
	return httpcore.Execute(ctx, s.roleB(resty.MethodDelete, opt, role), jsonmodels.NewTenant)
}

func (s *Service) roleB(method string, opt Options, role string) *httpcore.TryRequest {
	req := s.svc.Client.R().
		SetMethod(method).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(core.PathParamTenantID, opt.Tenant).
		SetPathParam(paramRoleID, role).
		SetURL(apiRestrictedRole)
	return httpcore.NewTryRequest(s.svc.Client, req)
}

func parseRoles(b []byte) []string {
	var roles []string
	if err := json.Unmarshal(b, &roles); err == nil {
		return roles
	}
	// Tolerate an object wrapper, e.g. {"restrictedRoles": [...]}.
	var wrapped map[string][]string
	if err := json.Unmarshal(b, &wrapped); err == nil {
		for _, v := range wrapped {
			return v
		}
	}
	return nil
}
