package c8y

// PaginationOptions is the cumulocity pagination options
type PaginationOptions struct {
	// Pagesize of results to return in one request
	PageSize int `url:"pageSize,omitempty"`

	// Include total pages included in the pagination at the given page size
	WithTotalPages bool `url:"withTotalPages,omitempty"`

	// Include count of elements in the statistics response. Only supported >= 10.13
	WithTotalElements bool `url:"withTotalElements,omitempty"`
}

// NewPaginationOptions returns a pagination options object with a specified pagesize and WithTotalPages set to false
func NewPaginationOptions(pageSize int) *PaginationOptions {
	return &PaginationOptions{
		PageSize: pageSize,
	}
}
