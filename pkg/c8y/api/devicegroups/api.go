// Package devicegroups provides access to Cumulocity device groups. A device
// group is a managed object carrying the c8y_IsDeviceGroup fragment, so this
// service wraps the managed-objects API and scopes reads and name resolution
// to that fragment — a plain managed-object lookup would also match ordinary
// devices.
package devicegroups

import (
	"context"
	"fmt"
	"strings"

	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/contexthelpers"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/core"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/model"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/types"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/jsonmodels"
	"github.com/reubenmiller/go-c8y/v2/pkg/c8y/op"
	"resty.dev/v3"
)

const (
	// FragmentIsDeviceGroup marks a managed object as a device group.
	FragmentIsDeviceGroup = "c8y_IsDeviceGroup"
	// TypeDeviceGroup is the type of a root device group.
	TypeDeviceGroup = "c8y_DeviceGroup"
	// TypeDeviceSubGroup is the type of a nested device group.
	TypeDeviceSubGroup = "c8y_DeviceSubGroup"
)

// ManagedObjectIterator iterates device groups.
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// GetOptions and DeleteOptions reuse the managed-object options.
type (
	GetOptions    = managedobjects.GetOptions
	DeleteOptions = managedobjects.DeleteOptions
)

// Service interacts with device groups.
type Service struct {
	core.Service

	managedObjects *managedobjects.Service
}

// NewService creates a device-groups service.
func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
	}
}

// GroupRef is a typed reference to a device group, resolved by ResolveID.
// Construct it with ByID or ByName, or cast a dynamic string with
// GroupRef(value).
type GroupRef string

// ByID creates a direct-ID reference (no lookup).
func ByID(id string) GroupRef { return GroupRef(id) }

// ByName creates a reference resolved by group name (wildcards allowed).
func ByName(name string) GroupRef { return GroupRef("name:" + name) }

// ListOptions filters device groups. The Query field is the fully-built q
// expression (e.g. from model.InventoryQuery); restrict it to device groups
// with ScopeToGroups when the caller has not already done so.
type ListOptions struct {
	Type         string `url:"type,omitempty"`
	FragmentType string `url:"fragmentType,omitempty"`
	Query        string `url:"q"`

	managedobjects.GetOptions
	pagination.PaginationOptions
}

// List returns a page of managed objects matching opt's query. It is thin (the
// query is passed through as-is, like devices.List); the resolver and
// ScopeToGroups are responsible for restricting results to device groups.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator over every device group matching opts. By default
// it uses the _id keyset optimisation; set ListOptions.Strategy to override.
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	strategy, err := managedobjects.ResolveListStrategy(opts.Strategy, opts.Query)
	if err != nil {
		return pagination.NewErrorIterator[jsonmodels.ManagedObject](err)
	}
	return pagination.PaginateWith(
		ctx,
		pagination.PageRequest{PaginationOptions: opts.PaginationOptions},
		strategy,
		func(req pagination.PageRequest) op.Result[jsonmodels.ManagedObject] {
			o := opts
			o.PaginationOptions = req.PaginationOptions
			if req.AfterID != "" {
				o.Query = model.WithIDCursor(opts.Query, req.AfterID)
			}
			return s.List(ctx, o)
		},
		jsonmodels.NewManagedObject,
	)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// Get returns a device group by reference (id, name:, or query:).
func (s *Service) Get(ctx context.Context, ref GroupRef, opt GetOptions) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Get(ctx, id, opt)
}

// Update updates a device group by reference.
func (s *Service) Update(ctx context.Context, ref GroupRef, body any) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Update(ctx, id, body)
}

// Delete deletes a device group by reference.
func (s *Service) Delete(ctx context.Context, ref GroupRef, opt DeleteOptions) op.Result[core.NoContent] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}
	return s.managedObjects.Delete(ctx, id, opt)
}

// Create creates a device group. The c8y_IsDeviceGroup fragment and a default
// type (c8y_DeviceGroup) are added when the body does not already set them.
func (s *Service) Create(ctx context.Context, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	if body == nil {
		body = map[string]any{}
	}
	if _, ok := body[FragmentIsDeviceGroup]; !ok {
		body[FragmentIsDeviceGroup] = map[string]any{}
	}
	if _, ok := body["type"]; !ok {
		body["type"] = TypeDeviceGroup
	}
	return s.managedObjects.Create(ctx, body)
}

// ResolveID resolves a device-group reference to an id. A plain string is used
// as-is; "name:<name>" and "query:<q>" are looked up scoped to device groups.
func (s *Service) ResolveID(ctx context.Context, ref string, meta map[string]any) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty device group reference")
	}
	scheme, value, hasScheme := strings.Cut(ref, ":")
	if !hasScheme {
		return ref, nil
	}
	// Resolve for real even under dry run / deferred execution.
	ctx = contexthelpers.ResolutionContext(ctx)
	switch scheme {
	case "id":
		return value, nil
	case "name":
		return s.lookupByQuery(ctx, model.NewInventoryQuery().
			HasFragment(FragmentIsDeviceGroup).
			AddFilterEqStr("name", value).
			AddOrderBy("name").
			Build(), value, meta)
	case "query":
		return s.lookupByQuery(ctx, ScopeToGroups(value), value, meta)
	default:
		return "", fmt.Errorf("unknown device group resolver scheme: %q", scheme)
	}
}

// lookupByQuery returns the id of the first device group matching query.
// label is the original reference, for error messages and metadata.
func (s *Service) lookupByQuery(ctx context.Context, query, label string, meta map[string]any) (string, error) {
	opt := ListOptions{Query: query}
	opt.PageSize = 1
	result := s.List(ctx, opt)
	if result.Err != nil {
		return "", result.Err
	}
	for item := range op.Iter(result) {
		if meta != nil {
			meta["id"] = item.ID()
			meta["name"] = item.Name()
			meta["source"] = "name"
		}
		return item.ID(), nil
	}
	return "", fmt.Errorf("device group not found: %s", label)
}

// ScopeToGroups builds a query that restricts results to device groups,
// combining the c8y_IsDeviceGroup fragment check with the given raw filter
// expression (not an already-built $filter string). An empty filter yields the
// bare fragment check.
func ScopeToGroups(filter string) string {
	fragment := fmt.Sprintf("has(%s)", FragmentIsDeviceGroup)
	return model.NewInventoryQuery().AddFilterPart(fragment, filter).Build()
}
