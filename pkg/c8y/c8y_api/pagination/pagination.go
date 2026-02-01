package pagination

import (
	"strings"
)

type PageSize int

const DefaultPageSize PageSize = 5

func (p PageSize) IsZero() bool {
	return p < PageSize(1) || p == PageSize(DefaultPageSize)
}

// PaginationOptions is the cumulocity pagination options
type PaginationOptions struct {
	// Pagesize of results to return in one request
	PageSize int `url:"pageSize,omitempty,omitzero" json:"pageSize,omitempty,omitzero"`

	// Include total pages included in the pagination at the given page size
	WithTotalPages bool `url:"withTotalPages,omitempty"`

	// Include count of elements in the statistics response. Only supported >= 10.13
	WithTotalElements bool `url:"withTotalElements,omitempty"`

	// Defines the slice of data to be returned, starting with 1. By default, the first page is returned.
	CurrentPage int `url:"currentPage,omitempty,omitzero"`

	// Limit to the maximum number of items when doing client side paging
	MaxItems int64 `url:"-"`
}

func (o PaginationOptions) IsZero() bool {
	return o.PageSize <= 0 || o.PageSize == 5 // Define zero as any non-positive value
}

func (o PaginationOptions) GetMaxItems() int64 {
	return o.MaxItems
}

// OptimalPageSize calculates the optimal page size based on MaxItems
// Returns the smaller of: MaxItems (if set), current PageSize (if set), or 2000 (max allowed)
func (o PaginationOptions) OptimalPageSize() int {
	const maxAllowed = 2000

	// If PageSize is already set, respect it but cap at max
	if o.PageSize > 0 {
		if o.PageSize > maxAllowed {
			return maxAllowed
		}
		return o.PageSize
	}

	// If MaxItems is set and less than max, use it as PageSize
	if o.MaxItems > 0 && o.MaxItems < maxAllowed {
		return int(o.MaxItems)
	}

	// Default to max allowed
	return maxAllowed
}

// Set the current page to return
func (o PaginationOptions) SetCurrentPage(v int) *PaginationOptions {
	o.CurrentPage = v
	return &o
}
func (o PaginationOptions) SetPageSize(v int) *PaginationOptions {
	o.PageSize = v
	return &o
}

type PagerOptions struct {
	MaxPages    int64 `url:"-"`
	MaxItems    int64 `url:"-"`
	PageSize    int64 `url:"pageSize"`
	CurrentPage int64 `url:"currentPage"`
}

func IncludeAll() PagerOptions {
	return PagerOptions{}
}

func DefaultSearch() PagerOptions {
	return PagerOptions{
		MaxItems: 6000,
	}
}

func (p *PagerOptions) GetPageSize() int64 {
	if p.PageSize <= 0 {
		return 2000
	}
	return p.PageSize
}

// NewPaginationOptions returns a pagination options object with a specified pagesize and WithTotalPages set to false
func NewPaginationOptions(pageSize int) PaginationOptions {
	return PaginationOptions{
		PageSize: pageSize,
	}
}

func trimHost(v string) string {
	i := strings.Index(v, "://") + 3
	for i < len(v) {
		if v[i] == '/' {
			break
		}
		i++
	}
	return v[i:]
}
