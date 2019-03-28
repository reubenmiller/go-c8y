package c8y

// PaginationOptions is the cumulocity pagination options
type PaginationOptions struct {
	// Pagesize of results to return in one request
	PageSize int `url:"pageSize,omitempty"`

	WithTotalPages bool `url:"withTotalPages,omitempty"`
}

// NewPaginationOptions returns a pagination options object with a specified pagesize and WithTotalPages set to false
func NewPaginationOptions(pageSize int) *PaginationOptions {
	return &PaginationOptions{
		PageSize: pageSize,
	}
}
