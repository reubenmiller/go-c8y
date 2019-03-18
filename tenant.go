package c8y

import (
	"context"
)

// TenantService does something
type TenantService service

// TenantSummaryOptions todo
type TenantSummaryOptions struct {
	DateFrom string `url:"dateFrom,omitempty"`
	DateTo   string `url:"dateTill,omitempty"`
}

// TenantSummary todo
type TenantSummary struct {
	Self                    string   `json:"self"`
	Day                     string   `json:"day"`
	DeviceCount             int      `json:"deviceCount"`
	DeviceWithChildrenCount int      `json:"deviceWithChildrenCount"`
	DeviceEndpointCount     int      `json:"deviceEndpointCount"`
	DeviceRequestCount      int      `json:"deviceRequestCount"`
	RequestCount            int      `json:"requestCount"`
	StorageSize             int      `json:"storageSize"`
	SubscribedApplications  []string `json:"subscribedApplications"`
}

// CurrentTenant todo
type CurrentTenant struct {
	AllowCreateTenants bool     `json:"allowCreateTenants"`
	CustomProperties   struct{} `json:"customProperties"`
	Name               string   `json:"name"`
}

// GetTenantSummary returns summary of requests and database usage from the start of this month until now.
func (s *TenantService) GetTenantSummary(ctx context.Context, opt *TenantSummaryOptions) (*TenantSummary, *Response, error) {
	data := new(TenantSummary)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/statistics/summary",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}
