// Package deviceprofiles provides access to Cumulocity device profiles. A device
// profile is a managed object of type c8y_Profile carrying the c8y_DeviceProfile
// and c8y_Filter fragments, so this service wraps the managed-objects API and
// scopes reads and name resolution to that type — a plain managed-object lookup
// would also match ordinary devices.
package deviceprofiles

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
	// TypeProfile is the type of a device profile managed object.
	TypeProfile = "c8y_Profile"
	// FragmentDeviceProfile holds the device profile definition.
	FragmentDeviceProfile = "c8y_DeviceProfile"
	// FragmentFilter restricts which devices a profile may be applied to.
	FragmentFilter = "c8y_Filter"
)

// ManagedObjectIterator iterates device profiles.
type ManagedObjectIterator = pagination.Iterator[jsonmodels.ManagedObject]

// GetOptions and DeleteOptions reuse the managed-object options.
type (
	GetOptions    = managedobjects.GetOptions
	DeleteOptions = managedobjects.DeleteOptions
)

// Service interacts with device profiles.
type Service struct {
	core.Service

	managedObjects *managedobjects.Service
}

// NewService creates a device-profiles service.
func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
	}
}

// DeviceProfileRef is a typed reference to a device profile, resolved by
// ResolveID. Construct it with ByID or ByName, or cast a dynamic string with
// DeviceProfileRef(value).
type DeviceProfileRef string

// ByID creates a direct-ID reference (no lookup).
func ByID(id string) DeviceProfileRef { return DeviceProfileRef(id) }

// ByName creates a reference resolved by device-profile name (wildcards allowed).
func ByName(name string) DeviceProfileRef { return DeviceProfileRef("name:" + name) }

// ListOptions filters device profiles. The Query field is the fully-built q
// expression (e.g. from model.InventoryQuery); restrict it to device profiles
// with ScopeToDeviceProfiles when the caller has not already done so.
type ListOptions struct {
	Type         string `url:"type,omitempty"`
	FragmentType string `url:"fragmentType,omitempty"`
	Query        string `url:"q"`

	managedobjects.GetOptions
	pagination.PaginationOptions
}

// List returns a page of managed objects matching opt's query. It is thin (the
// query is passed through as-is, like devices.List); the resolver and
// ScopeToDeviceProfiles are responsible for restricting results to device
// profiles.
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteCollection(ctx, s.listB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator over every device profile matching opts. By
// default it uses the _id keyset optimisation; set ListOptions.Strategy to
// override.
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

// Get returns a device profile by reference (id, name:, or query:).
func (s *Service) Get(ctx context.Context, ref DeviceProfileRef, opt GetOptions) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Get(ctx, id, opt)
}

// Update updates a device profile by reference.
func (s *Service) Update(ctx context.Context, ref DeviceProfileRef, body any) op.Result[jsonmodels.ManagedObject] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[jsonmodels.ManagedObject](err, false)
	}
	return s.managedObjects.Update(ctx, id, body)
}

// Delete deletes a device profile by reference.
func (s *Service) Delete(ctx context.Context, ref DeviceProfileRef, opt DeleteOptions) op.Result[core.NoContent] {
	id, err := s.ResolveID(ctx, string(ref), nil)
	if err != nil {
		return op.Failed[core.NoContent](err, false)
	}
	return s.managedObjects.Delete(ctx, id, opt)
}

// Create creates a device-profile managed object. The c8y_Profile type and the
// c8y_DeviceProfile / c8y_Filter fragments are added when the body (a map) does
// not already set them; other body shapes are passed through unchanged (the
// caller is responsible for including them).
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	if m, ok := body.(map[string]any); ok {
		if _, ok := m["type"]; !ok {
			m["type"] = TypeProfile
		}
		if _, ok := m[FragmentDeviceProfile]; !ok {
			m[FragmentDeviceProfile] = map[string]any{}
		}
		if _, ok := m[FragmentFilter]; !ok {
			m[FragmentFilter] = map[string]any{}
		}
		body = m
	}
	return s.managedObjects.Create(ctx, body)
}

// ResolveID resolves a device-profile reference to an id. A plain string is used
// as-is; "name:<name>" and "query:<q>" are looked up scoped to device profiles.
func (s *Service) ResolveID(ctx context.Context, ref string, meta map[string]any) (string, error) {
	if ref == "" {
		return "", fmt.Errorf("empty device profile reference")
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
			AddFilterEqStr("type", TypeProfile).
			AddFilterEqStr("name", value).
			AddOrderBy("name").
			Build(), value, meta)
	case "query":
		return s.lookupByQuery(ctx, ScopeToDeviceProfiles(value), value, meta)
	default:
		return "", fmt.Errorf("unknown device profile resolver scheme: %q", scheme)
	}
}

// lookupByQuery returns the id of the first device profile matching query. label
// is the original reference, for error messages and metadata.
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
	return "", fmt.Errorf("device profile not found: %s", label)
}

// ScopeToDeviceProfiles builds a query that restricts results to device
// profiles, combining the c8y_Profile type check with the given raw filter
// expression (not an already-built $filter string). An empty filter yields the
// bare type check.
func ScopeToDeviceProfiles(filter string) string {
	typeFilter := fmt.Sprintf("type eq '%s'", TypeProfile)
	return model.NewInventoryQuery().AddFilterPart(typeFilter, filter).Build()
}
