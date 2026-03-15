package child

import "github.com/reubenmiller/go-c8y/v2/pkg/c8y/api/pagination"

type ListOptions struct {
	Query             string `url:"query,omitempty"`
	WithParents       bool   `url:"withParents,omitempty"`
	WithChildren      bool   `url:"withChildren,omitempty"`
	WithChildrenCount bool   `url:"withChildrenCount,omitempty"`

	// Pagination options
	pagination.PaginationOptions
}
