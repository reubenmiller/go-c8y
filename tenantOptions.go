package c8y

import (
	"context"
)

// TenantOptionsService does something
type TenantOptionsService service

// TenantOption is a setting used to customise a tenant
type TenantOption struct {
	Category string `json:"category,omitempty"`
	Key      string `json:"key,omitempty"`
	Value    string `json:"value,omitempty"`
}

// TenantOptionsCollection todo
type TenantOptionsCollection struct {
	*BaseResponse

	Options []TenantOption `json:"options"`
}

// GetOptions returns collection of tenant options
func (s *TenantOptionsService) GetOptions(ctx context.Context, opt *PaginationOptions) (*TenantOptionsCollection, *Response, error) {
	data := new(TenantOptionsCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/options",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetOptionsForCategory returns collection of tenant options for the specified category
func (s *TenantOptionsService) GetOptionsForCategory(ctx context.Context, category string) (map[string]string, *Response, error) {
	data := make(map[string]string)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/options/" + category,
		ResponseData: &data,
	})
	return data, resp, err
}

// UpdateOptions updates multiple options for the specified category
func (s *TenantOptionsService) UpdateOptions(ctx context.Context, category string, body map[string]string) (map[string]string, *Response, error) {
	data := make(map[string]string)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "tenant/options/" + category,
		Body:         body,
		ResponseData: &data,
	})
	return data, resp, err
}

// UpdateEditability sets the editability of the given option. Only possible from management tenant
func (s *TenantOptionsService) UpdateEditability(ctx context.Context, category, key string, editable bool) (*TenantOption, *Response, error) {
	data := new(TenantOption)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "PUT",
		Path:   "tenant/options/" + category + "/" + key + "/editable",
		Body: map[string]bool{
			"editable": editable,
		},
		ResponseData: data,
	})
	return data, resp, err
}

// GetOption returns the given tenant option by category and key
func (s *TenantOptionsService) GetOption(ctx context.Context, category, key string) (*TenantOption, *Response, error) {
	data := new(TenantOption)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/options/" + category + "/" + key,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes an existing tenant option by category and key
func (s *TenantOptionsService) Delete(ctx context.Context, category, key string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "tenant/options/" + category + "/" + key,
	})
}

// Create adds a new tenant
func (s *TenantOptionsService) Create(ctx context.Context, body *TenantOption) (*TenantOption, *Response, error) {
	data := new(TenantOption)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "tenant/options",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update updates an existing tenant option
func (s *TenantOptionsService) Update(ctx context.Context, category, key string, value string) (*TenantOption, *Response, error) {
	data := new(TenantOption)

	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method: "PUT",
		Path:   "tenant/options/" + category + "/" + key,
		Body: TenantOption{
			Value: value,
		},
		ResponseData: data,
	})
	return data, resp, err
}

// GetSystemOptions returns collection system options
func (s *TenantOptionsService) GetSystemOptions(ctx context.Context, opt *PaginationOptions) (*TenantOptionsCollection, *Response, error) {
	data := new(TenantOptionsCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/system/options",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetSystemOption returns the given system option by category and key
func (s *TenantOptionsService) GetSystemOption(ctx context.Context, category, key string) (*TenantOption, *Response, error) {
	data := new(TenantOption)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/system/options/" + category + "/" + key,
		ResponseData: data,
	})
	return data, resp, err
}
