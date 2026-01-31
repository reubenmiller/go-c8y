package devices

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/inventory/managedobjects"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/pagination"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/types"
	"resty.dev/v3"
)

var ApiDeviceControlAccessToken = "/devicecontrol/deviceAccessToken"

const ParamId = "id"

const ResultProperty = managedobjects.ResultProperty

func NewService(s *core.Service) *Service {
	return &Service{
		Service:        *s,
		managedObjects: *managedobjects.NewService(s),
	}
}

// Service inventory api to interact with managed objects
// type Service core.Service
type Service struct {
	core.Service

	managedObjects managedobjects.Service
}

// ListOptions filter managed object
type ListOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	Text string `url:"text,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"q,omitempty"`

	managedobjects.GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// ManagedObjectIterator provides iteration over managed objects
type ManagedObjectIterator struct {
	items iter.Seq[jsonmodels.ManagedObject]
	err   error
}

func (it *ManagedObjectIterator) Items() iter.Seq[jsonmodels.ManagedObject] {
	return it.items
}

func (it *ManagedObjectIterator) Err() error {
	return it.err
}

func paginateManagedObjects(ctx context.Context, fetch func(page int) op.Result[jsonmodels.ManagedObject], maxItems int64) *ManagedObjectIterator {
	iterator := &ManagedObjectIterator{}

	iterator.items = func(yield func(jsonmodels.ManagedObject) bool) {
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
				item := jsonmodels.NewManagedObject(doc.Bytes())
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

// List managed objects
func (s *Service) List(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

// ListAll returns an iterator for all devices
func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	if opts.PageSize == 0 {
		opts.PageSize = 2000
	}
	return paginateManagedObjects(ctx, func(page int) op.Result[jsonmodels.ManagedObject] {
		opts.CurrentPage = page
		return s.List(ctx, opts)
	}, opts.GetMaxItems())
}

func (s *Service) ListB(opt ListOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}

// FindOptions filter devices
type FindOptions struct {
	Type string `url:"type,omitempty"`

	FragmentType string `url:"fragmentType,omitempty"`

	// Read-only collection of managed objects fetched for a given list of ids (placeholder {ids}),for example "?ids=41,43,68".
	Ids []string `url:"ids,omitempty"`

	Query string `url:"q,omitempty"`

	managedobjects.GetOptions

	// Pagination options
	pagination.PaginationOptions
}

// List managed objects
func (s *Service) Find(ctx context.Context, opt FindOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.FindB(opt), managedobjects.ResultProperty, types.ResponseFieldStatistics, jsonmodels.NewManagedObject)
}

func (s *Service) FindB(opt FindOptions) *core.TryRequest {
	req := s.Client.R().
		SetMethod(resty.MethodGet).
		SetQueryParamsFromValues(core.QueryParameters(opt)).
		SetURL(managedobjects.ApiManagedObjects)
	return core.NewTryRequest(s.Client, req, managedobjects.ResultProperty)
}
