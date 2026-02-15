package model

// Statistics statistics about the returned rest response
type Statistics struct {
	CurrentPage   int `json:"currentPage,omitzero"`
	PageSize      int `json:"pageSize,omitzero"`
	TotalPages    int `json:"totalPages,omitzero"`
	TotalElements int `json:"totalElements,omitzero"`
}

// BaseResponse common response from all c8y api calls
type BaseResponse struct {
	Next       string     `json:"next,omitempty"`
	Self       string     `json:"self,omitempty"`
	Statistics Statistics `json:"statistics,omitempty"`
}
