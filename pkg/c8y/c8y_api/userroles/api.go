package userroles

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/userroles/usergroups"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/userroles/users"
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
type RoleIterator struct {
	items iter.Seq[jsonmodels.Role]
	err   error
}

func (it *RoleIterator) Items() iter.Seq[jsonmodels.Role] {
	return it.items
}

func (it *RoleIterator) Err() error {
	return it.err
}

func paginateRoles(ctx context.Context, fetch func(page int) op.Result[jsonmodels.Role], maxItems int64) *RoleIterator {
	iterator := &RoleIterator{}

	iterator.items = func(yield func(jsonmodels.Role) bool) {
		page := 1
		count := int64(0)
		for {
			result := fetch(page)
			if result.Err != nil {
				iterator.err = result.Err
				return
			}
			countBeforeResults := count
			for doc := range result.Data.Iter() {
				if maxItems > 0 && count >= maxItems {
					return
				}
				item := jsonmodels.NewRole(doc.Bytes())
				if !yield(item) {
					return
				}
				count++
			}
			if countBeforeResults == count {
				slog.Info("Stopping pagination as results array is empty")
				return
			}

			totalPages, ok := result.Meta["totalPages"].(int64)
			if ok && page >= int(totalPages) {
				return
			}
			page++
		}
	}

	return iterator
}

// Retrieve all user roles in the tenant
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Role] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewRole)
}

// ListAll returns an iterator for all user roles
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *RoleIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateRoles(ctx, func(page int) op.Result[jsonmodels.Role] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
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
	return core.ExecuteReturnResult(ctx, s.GetB(opt), jsonmodels.NewRole)
}

func (s *Service) GetB(opt GetOption) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetPathParam(ParamName, opt.Name).
		SetURL(ApiRole)
	return core.NewTryRequest(s.Client, req)
}
