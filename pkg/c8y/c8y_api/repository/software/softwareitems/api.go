package softwareitems

import (
	"context"
	"fmt"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiManagedObjects = "/inventory/managedObjects"
var ApiManagedObject = "/inventory/managedObjects/{id}"

const ParamId = "id"

const ResultProperty = "managedObjects"

const FragmentSoftware = "c8y_Software"
const FragmentSoftwareBinary = "c8y_SoftwareBinary"

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: managedobjects.NewService(s),
	}
}

// Service api to interact with software items
type Service struct {
	core.Service
	managedObjects *managedobjects.Service
}

// Create a software item
func (s *Service) Create(ctx context.Context, body any) op.Result[jsonmodels.Software] {
	return core.ExecuteReturnResult(ctx, s.createB(body), jsonmodels.NewSoftware)
}

// ListOptions filter software
type ListOptions struct {
	Name string `url:"-"`

	SoftwareType string `url:"-"`

	DeviceType string `url:"-"`

	// Pagination options
	pagination.PaginationOptions
}

// List software
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.Software] {
	return core.ExecuteReturnCollection(ctx, s.listB(opt), ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewSoftware)
}

func (s *Service) listB(opt ListOptions) *core.TryRequest {
	// Build request directly since managedObjects.listB is now private
	listOpts := managedobjects.ListOptions{
		Query: model.NewInventoryQuery().
			AddOrderBy("name").
			AddOrderBy("creationTime").
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterEqStr("name", opt.Name).
			AddFilterEqStr("softwareType", opt.SoftwareType).
			AddFilterEqStr("c8y_Filter.type", opt.DeviceType).
			Build(),
		PaginationOptions: pagination.PaginationOptions{
			CurrentPage: opt.CurrentPage,
			PageSize:    opt.PageSize,
		},
	}
	// Build the request directly
	req := s.managedObjects.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(listOpts)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.managedObjects.Client, req, managedobjects.ResultProperty)
}

// SoftwareIterator provides iteration over software items
type SoftwareIterator = pagination.Iterator[jsonmodels.Software]

// ListAll returns an iterator for all software items
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *SoftwareIterator {
	return pagination.Paginate(ctx, opts.PaginationOptions, func() op.Result[jsonmodels.Software] {
		return s.List(ctx, opts)
	}, jsonmodels.NewSoftware)
}

type GetOptions struct {
	// Lookup strategies (at least one required)
	ID           string `url:"-"`
	Name         string `url:"-"`
	SoftwareType string `url:"-"` // Used with Name for more specific lookup

	// Query options
	WithParents       bool `url:"withParents,omitempty"`
	WithChildren      bool `url:"withChildren,omitempty"`
	withChildrenCount bool `url:"withChildrenCount,omitempty"`
	SkipChildrenNames bool `url:"skipChildrenNames,omitempty"`
}

// UpdateOptions options for updating a software item
type UpdateOptions struct {
	// Lookup strategies (at least one required)
	ID           string `url:"-"`
	Name         string `url:"-"`
	SoftwareType string `url:"-"` // Used with Name for more specific lookup
}

// DeleteOptions options to delete a software item
type DeleteOptions struct {
	// Lookup strategies (at least one required)
	ID           string `url:"-"`
	Name         string `url:"-"`
	SoftwareType string `url:"-"` // Used with Name for more specific lookup

	// Delete options
	SkipCascade bool `url:"-"`
}

// resolveID resolves a software ID from various lookup strategies
func (s *Service) resolveID(ctx context.Context, id, name, softwareType string) (string, op.Result[jsonmodels.Software]) {
	// Direct ID provided
	if id != "" {
		return id, op.Result[jsonmodels.Software]{}
	}

	// Lookup by name (and optional softwareType)
	if name != "" {
		query := model.NewInventoryQuery().
			AddFilterEqStr("type", FragmentSoftware).
			AddFilterEqStr("name", name).
			AddFilterEqStr("softwareType", softwareType).
			Build()

		listResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if listResult.Err != nil {
			return "", op.Failed[jsonmodels.Software](
				fmt.Errorf("failed to lookup software by name: %w", listResult.Err),
				true,
			)
		}

		// Check if any items were found
		for item := range listResult.Data.Iter() {
			found := jsonmodels.NewSoftware(item.Bytes())
			return found.ID(), op.Result[jsonmodels.Software]{}
		}

		return "", op.Failed[jsonmodels.Software](
			fmt.Errorf("software not found with name=%s, softwareType=%s", name, softwareType),
			false,
		)
	}

	// No lookup strategy provided
	return "", op.Failed[jsonmodels.Software](
		fmt.Errorf("no lookup strategy provided: must specify ID or Name"),
		false,
	)
}

// Get a software item
func (s *Service) Get(ctx context.Context, opt GetOptions) op.Result[jsonmodels.Software] {
	id, resolveResult := s.resolveID(ctx, opt.ID, opt.Name, opt.SoftwareType)
	if resolveResult.Err != nil {
		return resolveResult
	}

	result := core.ExecuteReturnResult(ctx, s.getB(id, opt), jsonmodels.NewSoftware)

	// Add lookup metadata
	if opt.ID != "" {
		result.Meta["lookupMethod"] = "id"
	} else if opt.Name != "" {
		result.Meta["lookupMethod"] = "name"
		result.Meta["lookupName"] = opt.Name
		if opt.SoftwareType != "" {
			result.Meta["lookupSoftwareType"] = opt.SoftwareType
		}
	}

	return result
}

// Update a software item
func (s *Service) Update(ctx context.Context, opt UpdateOptions, body any) op.Result[jsonmodels.Software] {
	id, resolveResult := s.resolveID(ctx, opt.ID, opt.Name, opt.SoftwareType)
	if resolveResult.Err != nil {
		return resolveResult
	}

	result := core.ExecuteReturnResult(ctx, s.updateB(id, body), jsonmodels.NewSoftware)

	// Add lookup metadata
	if opt.ID != "" {
		result.Meta["lookupMethod"] = "id"
	} else if opt.Name != "" {
		result.Meta["lookupMethod"] = "name"
		result.Meta["lookupName"] = opt.Name
	}

	return result
}

// Delete a software item
func (s *Service) Delete(ctx context.Context, opt DeleteOptions) op.Result[jsonmodels.Software] {
	id, resolveResult := s.resolveID(ctx, opt.ID, opt.Name, opt.SoftwareType)
	if resolveResult.Err != nil {
		return resolveResult
	}

	result := core.ExecuteReturnResult(ctx, s.deleteB(id, opt), jsonmodels.NewSoftware)

	// Add lookup metadata
	if opt.ID != "" {
		result.Meta["lookupMethod"] = "id"
	} else if opt.Name != "" {
		result.Meta["lookupMethod"] = "name"
		result.Meta["lookupName"] = opt.Name
	}

	return result
}

// GetOrCreateByName searches by name and optional software type, creating if not found
func (s *Service) GetOrCreateByName(ctx context.Context, name, softwareType string, body any) op.Result[jsonmodels.Software] {
	query := model.NewInventoryQuery().
		AddFilterEqStr("type", FragmentSoftware).
		AddFilterEqStr("name", name).
		AddFilterEqStr("softwareType", softwareType).
		AddOrderBy("name").
		AddOrderBy("creationTime").
		Build()
	return s.getOrCreateWithQuery(ctx, body, query)
}

// GetOrCreateWith provides a generic query-based lookup
// Example queries:
//   - "name eq 'MySoftware' and softwareType eq 'application'"
//   - "name eq 'MySoftware'"
func (s *Service) GetOrCreateWith(ctx context.Context, body any, query string) op.Result[jsonmodels.Software] {
	query_ := model.NewInventoryQuery().
		AddFilterEqStr("type", FragmentSoftware).
		AddFilterPart(query).
		AddOrderBy("name").
		AddOrderBy("creationTime").
		Build()
	return s.getOrCreateWithQuery(ctx, body, query_)
}

// getOrCreateWithQuery is the internal implementation
func (s *Service) getOrCreateWithQuery(ctx context.Context, body any, query string) op.Result[jsonmodels.Software] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.Software], bool) {
		searchOpts := ListOptions{}
		searchOpts.PageSize = 1
		// Build query with the search criteria
		moResult := s.managedObjects.List(ctx, managedobjects.ListOptions{
			Query: query,
			PaginationOptions: pagination.PaginationOptions{
				PageSize: 1,
			},
		})

		if moResult.Err != nil {
			return op.Result[jsonmodels.Software]{}, false
		}

		// Check if any items were found
		for item := range moResult.Data.Iter() {
			found := jsonmodels.NewSoftware(item.Bytes())
			result := op.OK(found)
			result.HTTPStatus = moResult.HTTPStatus
			result.Meta["lookupMethod"] = "query"
			result.Meta["query"] = query
			return result, true
		}

		// Not found
		return op.Result[jsonmodels.Software]{}, false
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.Software] {
		createResult := s.Create(ctx, body)
		if createResult.Err != nil {
			return createResult
		}
		return createResult
	}

	// Execute get-or-create pattern
	return op.GetOrCreateR(ctx, finder, creator)
}

// Builder methods for backwards compatibility and flexibility

func (s *Service) createB(body any) *core.TryRequest {
	req := s.Service.Client.R().
		SetMethod(resty.MethodPost).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObjects)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) getB(ID string, opt GetOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetPathParam(ParamId, ID).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) updateB(ID string, body any) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodPut).
		SetPathParam(ParamId, ID).
		SetBody(body).
		SetHeader("Accept", types.MimeTypeApplicationJSON).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}

func (s *Service) deleteB(ID string, opt DeleteOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodDelete).
		SetPathParam(ParamId, ID).
		SetQueryParam("forceCascade", fmt.Sprintf("%v", !opt.SkipCascade)).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(ApiManagedObject)
	return core.NewTryRequest(s.Client, req)
}
