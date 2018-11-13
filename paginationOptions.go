package go-c8y

// PaginationOptions is the cumulocity pagination options
type PaginationOptions struct {
	// Pagesize of results to return in one request
	PageSize int `url:"pageSize,omitempty"`

	WithTotalPages bool `url:"withTotalPages,omitempty"`
}
