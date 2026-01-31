package jsondoc

import (
	"context"
	"iter"

	"github.com/reubenmiller/go-c8y/pkg/c8y/c8y_api/alternative/op"
)

// PaginationConfig controls pagination behavior
type PaginationConfig struct {
	MaxItems    int // Maximum total items to fetch (0 = unlimited)
	MaxPages    int // Maximum pages to fetch (0 = unlimited)
	PageSize    int
	CurrentPage int
}

// PaginatedFetch is a function that fetches a page of results
// currentPage is 1-indexed
type PaginatedFetch func(ctx context.Context, currentPage int) op.Result[JSONDoc]

// PaginatedFetchGeneric is a generic function that fetches a page of results
type PaginatedFetchGeneric[TData any] func(ctx context.Context, currentPage int) op.Result[TData]

// IteratorExtractor extracts an iterator from the result data
type IteratorExtractor[TData any, TItem any] func(TData) iter.Seq[TItem]

// PaginateGeneric creates an iterator that automatically fetches pages as needed for any type
func PaginateGeneric[TData any, TItem any](
	ctx context.Context,
	fetch PaginatedFetchGeneric[TData],
	getIter IteratorExtractor[TData, TItem],
	config PaginationConfig,
) iter.Seq[TItem] {
	return func(yield func(TItem) bool) {
		currentPage := 1
		totalItems := 0

		for {
			// Check page limit
			if config.MaxPages > 0 && currentPage > config.MaxPages {
				return
			}

			// Fetch the page
			result := fetch(ctx, currentPage)
			if result.Err != nil {
				return
			}

			// Iterate over items in this page
			for item := range getIter(result.Data) {
				// Check item limit
				if config.MaxItems > 0 && totalItems >= config.MaxItems {
					return
				}

				if !yield(item) {
					return
				}
				totalItems++
			}

			// Check if there are more pages
			totalPages, ok := result.Meta["totalPages"].(int64)
			if !ok || currentPage >= int(totalPages) {
				return
			}

			currentPage++
		}
	}
}

// Paginate creates an iterator that automatically fetches pages as needed
func Paginate(ctx context.Context, fetch PaginatedFetch, config PaginationConfig) iter.Seq[JSONDoc] {
	genericFetch := func(ctx context.Context, page int) op.Result[JSONDoc] {
		return fetch(ctx, page)
	}
	return PaginateGeneric[JSONDoc, JSONDoc](ctx, genericFetch, func(data JSONDoc) iter.Seq[JSONDoc] {
		return data.Iter()
	}, config)
}

// PaginateAll fetches all pages without limits
func PaginateAll(ctx context.Context, fetch PaginatedFetch) iter.Seq[JSONDoc] {
	return Paginate(ctx, fetch, PaginationConfig{})
}

// PaginateLimit fetches up to maxItems across all pages
func PaginateLimit(ctx context.Context, fetch PaginatedFetch, maxItems int) iter.Seq[JSONDoc] {
	return Paginate(ctx, fetch, PaginationConfig{MaxItems: maxItems})
}
