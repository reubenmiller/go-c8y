package managedobjects

import (
	"context"
	"iter"
	"log/slog"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/jsonmodels"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/core"
)

func (s *Service) Create2(ctx context.Context, body any) op.Result[jsonmodels.ManagedObject] {
	return core.ExecuteReturnResult(ctx, s.CreateB(body), jsonmodels.NewManagedObject)
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
