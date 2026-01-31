package managedobjects

import (
	"context"
	"fmt"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/model"
)

func (s *Service) Create2(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewManagedObject)
}

func (s *Service) Get2(ctx context.Context, ID string, opt GetOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.GetB(ID, opt), jsonmodels.NewManagedObject)
}

func (s *Service) Update2(ctx context.Context, ID string, body any) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.UpdateB(ID, body), jsonmodels.NewManagedObject)
}

func (s *Service) Delete2(ctx context.Context, ID string, opt DeleteOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.DeleteB(ID, opt), jsonmodels.NewManagedObject)
}

// GetOrCreateByName searches by name and optionally type, creating if not found
func (s *Service) GetOrCreateByName(ctx context.Context, name, objType string, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	query := model.NewInventoryQuery().
		AddFilterEqStr("name", name).
		AddFilterEqStr("type", objType).
		Build()
	return s.getOrCreateWithQuery(ctx, body, query)
}

// GetOrCreateByFragment searches for objects with a specific fragment property
func (s *Service) GetOrCreateByFragment(ctx context.Context, fragment string, body map[string]any) op.Result[jsonmodels.ManagedObject] {
	if fragment == "" {
		return op.Failed[jsonmodels.ManagedObject](fmt.Errorf("fragment must be set"), false)
	}
	query := model.NewInventoryQuery().
		HasFragment(fragment).
		Build()
	return s.getOrCreateWithQuery(ctx, body, query)
}

// GetOrCreateWith provides a generic query-based lookup
// Example queries:
//   - "name eq 'device01' and type eq 'c8y_Device'"
//   - "has(c8y_IsDevice) and c8y_Serial eq '12345'"
//   - "fragmentType eq 'c8y_CustomFragment'"
func (s *Service) GetOrCreateWith(ctx context.Context, body map[string]any, query string) op.Result[jsonmodels.ManagedObject] {
	query_ := model.NewInventoryQuery().
		AddFilterPart(query).
		Build()
	return s.getOrCreateWithQuery(ctx, body, query_)
}

// getOrCreateWithQuery is the internal implementation
func (s *Service) getOrCreateWithQuery(ctx context.Context, body map[string]any, query string) op.Result[jsonmodels.ManagedObject] {
	// Define finder function
	finder := func(ctx context.Context) (op.Result[jsonmodels.ManagedObject], bool) {
		searchOpts := ListOptions{}
		searchOpts.PaginationOptions.PageSize = 1
		searchOpts.Query = query

		listResult := s.List2(ctx, searchOpts)
		if listResult.Err != nil {
			return listResult, false
		}

		// Check if any items were found
		for item := range listResult.Data.Iter() {
			found := jsonmodels.NewManagedObject(item.Bytes())
			result := op.NewOK(found)
			result.HTTPStatus = listResult.HTTPStatus
			result.RequestID = listResult.RequestID
			result.Meta["query"] = query
			return result, true
		}

		return op.Result[jsonmodels.ManagedObject]{}, false
	}

	// Define creator function
	creator := func(ctx context.Context) op.Result[jsonmodels.ManagedObject] {
		createResult := s.Create2(ctx, body)
		result := op.NewCreated(createResult.Data)
		result.Err = createResult.Err
		result.HTTPStatus = createResult.HTTPStatus
		result.RequestID = createResult.RequestID
		result.Meta["query"] = query
		return result
	}

	// Execute get-or-create pattern (automatically sets Meta["found"])
	return op.GetOrCreateR(ctx, finder, creator)
}

func (s *Service) List2(ctx context.Context, opt ListOptions) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnCollection(ctx, s.ListB(opt), "managedObjects", "statistics", jsonmodels.NewManagedObject)
}

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

func paginateManagedObjects(ctx context.Context, fetch func(page int) op.Result[jsonmodels.ManagedObject], maxItems int) *ManagedObjectIterator {
	iterator := &ManagedObjectIterator{}

	iterator.items = func(yield func(jsonmodels.ManagedObject) bool) {
		page := 1
		count := 0
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
				// No more results
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

func (s *Service) ListAll(ctx context.Context, opts ListOptions) *ManagedObjectIterator {
	return paginateManagedObjects(ctx, func(page int) op.Result[jsonmodels.ManagedObject] {
		opts.CurrentPage = page
		opts.PageSize = 2000
		return s.List2(ctx, opts)
	}, 0)
}

func (s *Service) ListLimit(ctx context.Context, opts ListOptions, maxItems int) *ManagedObjectIterator {
	return paginateManagedObjects(ctx, func(page int) op.Result[jsonmodels.ManagedObject] {
		opts.CurrentPage = page
		opts.PageSize = 2000
		return s.List2(ctx, opts)
	}, maxItems)
}
