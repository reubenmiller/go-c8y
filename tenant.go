package go-c8y

import (
	"context"
	"fmt"
	"log"
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
	u := fmt.Sprintf("tenant/statistics/summary")

	queryParams, err := addOptions("", opt)
	if err != nil {
		return nil, nil, err
	}

	req, err := s.client.NewRequest("GET", u, queryParams, nil)
	if err != nil {
		return nil, nil, err
	}

	summary := new(TenantSummary)

	resp, err := s.client.Do(ctx, req, summary)
	if err != nil {
		return nil, resp, err
	}

	println("JSONData: ", *resp.JSONData)
	log.Printf("Total count: %d\n", summary.StorageSize)

	return summary, resp, nil
}
