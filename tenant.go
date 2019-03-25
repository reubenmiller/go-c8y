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

type TenantUsageStatisticsCollection struct {
	*BaseResponse
	UsageStatistics []TenantSummary `json:"usageStatistics,omitempty"`
}

// CurrentTenant todo
type CurrentTenant struct {
	Name               string      `json:"name"`
	DomainName         string      `json:"domainName"`
	AllowCreateTenants bool        `json:"allowCreateTenants"`
	CustomProperties   interface{} `json:"customProperties"`
}

type TenantUsageStatisticsSummary struct {
	DeviceCount             int      `json:"deviceCount"`
	DeviceWithChildrenCount int      `json:"deviceWithChildrenCount"`
	DeviceEndpointCount     int      `json:"deviceEndpointCount"`
	DeviceRequestCount      int      `json:"deviceRequestCount"`
	RequestCount            int      `json:"requestCount"`
	StorageSize             int      `json:"storageSize"`
	SubscribedApplications  []string `json:"subscribedApplications"`
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

// GetTenantStatistics returns statics for the current tenant between the specified days
func (s *TenantService) GetTenantStatistics(ctx context.Context, opt *TenantSummaryOptions) (*TenantUsageStatisticsCollection, *Response, error) {
	data := new(TenantUsageStatisticsCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/statistics",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetAllTenantsSummary returns the usage statistics from all of the tenants
func (s *TenantService) GetAllTenantsSummary(ctx context.Context, opt *TenantSummaryOptions) ([]TenantUsageStatisticsSummary, *Response, error) {
	data := make([]TenantUsageStatisticsSummary, 0)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/statistics/allTenantsSummary",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetCurrentTenant returns tenant for the currently logged in service user's tenant
func (s *TenantService) GetCurrentTenant(ctx context.Context) (*CurrentTenant, *Response, error) {
	data := new(CurrentTenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/currentTenant",
		ResponseData: data,
	})
	return data, resp, err
}

type ApplicationReferenceCollection struct {
	*BaseResponse
	References []Application `json:"references,omitempty"`
}

// Tenant [application/vnd.com.nsn.cumulocity.tenant+json]
type Tenant struct {
	ID                     string                       `json:"id,omitempty"`
	Self                   string                       `json:"self,omitempty"`
	Status                 string                       `json:"status,omitempty"`
	AdminName              string                       `json:"adminName,omitempty"`
	AdminEmail             string                       `json:"adminEmail,omitempty"`
	AllowCreateTenants     bool                         `json:"allowCreateTenants,omitempty"`
	StorageLimitPerDevice  int64                        `json:"storageLimitPerDevice,omitempty"`
	AdminPassword          string                       `json:"adminPassword,omitempty"`
	SendPasswordResetEmail bool                         `json:"sendPasswordResetEmail,omitempty"`
	Domain                 string                       `json:"domain,omitempty"`
	Company                string                       `json:"company,omitempty"`
	ContactName            string                       `json:"contactName,omitempty"`
	ContactPhone           string                       `json:"contactPhone,omitempty"`
	Applications           []ApplicationTenantReference `json:"applications,omitempty"`
	OwnedApplications      []ApplicationTenantReference `json:"ownedApplications,omitempty"`
	CustomProperties       interface{}                  `json:"customProperties,omitempty"`
	Parent                 string                       `json:"parent,omitempty"`
}

// TenantCollection todo
type TenantCollection struct {
	*BaseResponse

	Tenants []Tenant `json:"tenants"`
}

type ApplicationReference struct {
	Self      string `json:"self,omitempty"`
	Reference string `json:"reference,omitempty"`
}

// GetTenants returns collection of tenants
func (s *TenantService) GetTenants(ctx context.Context, opt *PaginationOptions) (*TenantCollection, *Response, error) {
	data := new(TenantCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants",
		Query:        opt,
		ResponseData: data,
	})
	return data, resp, err
}

// GetTenant returns a tenant using its ID
func (s *TenantService) GetTenant(ctx context.Context, ID string) (*Tenant, *Response, error) {
	data := new(Tenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants/" + ID,
		ResponseData: data,
	})
	return data, resp, err
}

// Create adds a new tenant
func (s *TenantService) Create(ctx context.Context, body *Tenant) (*Tenant, *Response, error) {
	data := new(Tenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "tenant/tenants",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Update adds an existing tenant
func (s *TenantService) Update(ctx context.Context, ID string, body *Tenant) (*Tenant, *Response, error) {
	data := new(Tenant)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "PUT",
		Path:         "tenant/tenants/" + ID,
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// Delete removes a tenant and all of its data
func (s *TenantService) Delete(ctx context.Context, ID string, body *Tenant) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "tenant/tenants/" + ID,
	})
}

//
// Application Reference Collection
//

// AddApplicationReference adds a new tenant
func (s *TenantService) AddApplicationReference(ctx context.Context, tenantID string, body *ApplicationTenantReference) (*ApplicationReference, *Response, error) {
	data := new(ApplicationReference)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "POST",
		Path:         "tenant/tenants/" + tenantID + "/applications",
		Body:         body,
		ResponseData: data,
	})
	return data, resp, err
}

// GetApplicationReferences returns list of applications associated with the tenant
func (s *TenantService) GetApplicationReferences(ctx context.Context, tenantID string, opts *PaginationOptions) (*ApplicationReferenceCollection, *Response, error) {
	data := new(ApplicationReferenceCollection)
	resp, err := s.client.SendRequest(ctx, RequestOptions{
		Method:       "GET",
		Path:         "tenant/tenants/" + tenantID + "/applications",
		Query:        opts,
		ResponseData: data,
	})
	return data, resp, err
}

// DeleteApplicationReference removes an application references from the tenant
func (s *TenantService) DeleteApplicationReference(ctx context.Context, tenantID string, applicationID string) (*Response, error) {
	return s.client.SendRequest(ctx, RequestOptions{
		Method: "DELETE",
		Path:   "tenant/tenants/" + tenantID + "/applications/" + applicationID,
	})
}
